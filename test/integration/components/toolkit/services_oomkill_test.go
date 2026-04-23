package toolkit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

// TestServiceOOMKilledAfterReadiness_Integration verifies that a service which
// passes its readiness probe but subsequently gets OOMKilled is detected as a
// failure by the service monitor (processNotifications), rather than being
// silently marked as green/passed.
//
// This is a regression test for TKC-5014.
func TestServiceOOMKilledAfterReadiness_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	services := map[string]testworkflowsv1.ServiceSpec{
		"oom-after-ready": {
			IndependentServiceSpec: testworkflowsv1.IndependentServiceSpec{
				StepRun: testworkflowsv1.StepRun{
					ContainerConfig: testworkflowsv1.ContainerConfig{
						Image: "python:3.14.3-slim-trixie",
						Resources: &testworkflowsv1.Resources{
							Limits: map[corev1.ResourceName]intstr.IntOrString{
								corev1.ResourceMemory: intstr.FromString("50Mi"),
							},
						},
					},
					Shell: common.Ptr(
						"touch /tmp/ready && echo 'ready' && sleep 2 && python3 -c \"a = ' ' * 10**9\"",
					),
				},
				Pod:           &testworkflowsv1.PodConfig{},
				RestartPolicy: testworkflowsv1.ServiceRestartPolicyNever,
				ReadinessProbe: &corev1.Probe{
					PeriodSeconds:    1,
					SuccessThreshold: 1,
					FailureThreshold: 3,
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"sh", "-c", "test -f /tmp/ready"},
						},
					},
				},
			},
		},
	}

	err := executeServices(t, services, "test-group")

	assert.Error(t, err,
		"service OOMKilled after readiness should be detected as failure (TKC-5014)")
}

// TestServiceOOMKilledDetectedAtStop_Integration verifies that the stop/kill
// phase detects a service that was OOMKilled after readiness, even if the
// monitoring phase missed it. This is the safety-net for TKC-5014: the kill
// command checks service health before destroying resources.
func TestServiceOOMKilledDetectedAtStop_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	services := map[string]testworkflowsv1.ServiceSpec{
		"oom-after-ready": {
			IndependentServiceSpec: testworkflowsv1.IndependentServiceSpec{
				StepRun: testworkflowsv1.StepRun{
					ContainerConfig: testworkflowsv1.ContainerConfig{
						Image: "python:3.14.3-slim-trixie",
						Resources: &testworkflowsv1.Resources{
							Limits: map[corev1.ResourceName]intstr.IntOrString{
								corev1.ResourceMemory: intstr.FromString("50Mi"),
							},
						},
					},
					Shell: common.Ptr(
						"touch /tmp/ready && echo 'ready' && sleep 2 && python3 -c \"a = ' ' * 10**9\"",
					),
				},
				Pod:           &testworkflowsv1.PodConfig{},
				RestartPolicy: testworkflowsv1.ServiceRestartPolicyNever,
				ReadinessProbe: &corev1.Probe{
					PeriodSeconds:    1,
					SuccessThreshold: 1,
					FailureThreshold: 3,
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"sh", "-c", "test -f /tmp/ready"},
						},
					},
				},
			},
		},
	}

	groupRef := "test-group-stop"
	_ = executeServices(t, services, groupRef)

	// Wait for the OOMKill to actually happen before running the stop phase.
	pod := waitForPodInNamespace(t, namespace, 30*time.Second)
	require.NotNil(t, pod, "should find the service pod in the namespace")

	pod = waitForPodTermination(t, namespace, pod.Name, 60*time.Second)
	require.Equal(t, corev1.PodFailed, pod.Status.Phase,
		"pod should be in Failed phase after OOMKill")
	require.Equal(t, "OOMKilled", getContainerTerminatedReason(pod, "2"),
		"container should have been OOMKilled")

	cfg, err := config.LoadConfigV2()
	require.NoError(t, err)

	err = commands.RunKillWithOptions(context.Background(), cfg, groupRef)
	assert.Error(t, err,
		"kill command should detect OOMKilled service as a failure (TKC-5014 safety net)")
}

func waitForPodInNamespace(t *testing.T, namespace string, timeout time.Duration) *corev1.Pod {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		podList, err := globalK8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{Limit: 1})
		require.NoError(t, err)

		if len(podList.Items) > 0 {
			return &podList.Items[0]
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func getContainerTerminatedReason(pod *corev1.Pod, containerName string) string {
	for _, s := range pod.Status.ContainerStatuses {
		if s.Name == containerName && s.State.Terminated != nil {
			return s.State.Terminated.Reason
		}
	}
	for _, s := range pod.Status.InitContainerStatuses {
		if s.Name == containerName && s.State.Terminated != nil {
			return s.State.Terminated.Reason
		}
	}
	return ""
}
