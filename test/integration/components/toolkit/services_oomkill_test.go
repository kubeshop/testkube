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

// oomTestMemoryLimit is applied to every container in the service pod, including
// testkube's toolkit init container. 50Mi starved init (OOMKilled, exit 137,
// before the service could start); 256Mi comfortably fits init and the idle
// service, while the ~1GB allocation in the service shell still blows past it,
// so only the intended OOM happens.
const oomTestMemoryLimit = "256Mi"

// oomTriggerCmd allocates a string of 10**9 spaces. CPython stores ASCII as
// 1 byte/char, so this is a ~1GB resident allocation — far above
// oomTestMemoryLimit — which the kernel answers with an OOM kill of the
// container.
const oomTriggerCmd = `python3 -c "a = ' ' * 10**9"`

// oomReadyFile is the readiness sentinel: serviceReadyCmd creates it and the
// ReadinessProbe checks for it. One const so the shell and the probe can't
// drift apart.
const oomReadyFile = "/tmp/ready"

// serviceReadyCmd creates oomReadyFile so the ReadinessProbe passes and the
// monitor observes the service as ready.
const serviceReadyCmd = "touch " + oomReadyFile + " && echo 'ready'"

// oomAfterReadyShell is the command both OOM tests run: become ready, stay
// healthy a couple seconds so the monitor observes readiness, then allocate
// ~1GB and get OOMKilled. The OOM restarts the container after readiness; it is
// not a finished workflow, so the monitor (which detaches at readiness) still
// reports success.
const oomAfterReadyShell = serviceReadyCmd + " && sleep 2 && " + oomTriggerCmd

// oomService is the shared service definition for the OOM tests: a python
// container capped at oomTestMemoryLimit running oomAfterReadyShell, with a
// readiness probe that passes once serviceReadyCmd has run.
func oomService() map[string]testworkflowsv1.ServiceSpec {
	return map[string]testworkflowsv1.ServiceSpec{
		"oom-after-ready": {
			IndependentServiceSpec: testworkflowsv1.IndependentServiceSpec{
				StepRun: testworkflowsv1.StepRun{
					ContainerConfig: testworkflowsv1.ContainerConfig{
						Image: "python:3.14.3-slim-trixie",
						Resources: &testworkflowsv1.Resources{
							Limits: map[corev1.ResourceName]intstr.IntOrString{
								corev1.ResourceMemory: intstr.FromString(oomTestMemoryLimit),
							},
						},
					},
					Shell: common.Ptr(oomAfterReadyShell),
				},
				Pod: &testworkflowsv1.PodConfig{},
				ReadinessProbe: &corev1.Probe{
					PeriodSeconds:    1,
					SuccessThreshold: 1,
					FailureThreshold: 3,
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"sh", "-c", "test -f " + oomReadyFile},
						},
					},
				},
			},
		},
	}
}

// TestServiceOOMKilledAfterReadiness_Integration verifies that a service which
// passes its readiness probe but subsequently gets OOMKilled is marked as ready
// by the monitor (which stops after readiness), and the OOMKill is caught at
// stop/kill time.
func TestServiceOOMKilledAfterReadiness_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	groupRef := "test-group-monitor"
	// The service becomes ready, then OOMs a couple seconds later. An OOM is a
	// container restart, not a finished workflow, and the monitor detaches once
	// it observes readiness — so it must report success here. (The flake this
	// fixes was the toolkit init container OOMing at 50Mi, which kept the pod
	// from ever reaching readiness; oomTestMemoryLimit gives it room.)
	err := executeServices(t, oomService(), groupRef)
	require.NoError(t, err, "monitor should mark the service ready despite the later OOM")

	assertOOMDetectedAtKill(t, namespace, groupRef)
}

// TestServiceOOMKilledDetectedAtStop_Integration verifies that the stop/kill
// phase detects a service that was OOMKilled after readiness, even if the
// monitoring phase missed it. This is the safety net: the kill command checks
// service health before destroying resources.
func TestServiceOOMKilledDetectedAtStop_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	groupRef := "test-group-stop"
	_ = executeServices(t, oomService(), groupRef)

	assertOOMDetectedAtKill(t, namespace, groupRef)
}

// assertOOMDetectedAtKill waits for the service pod to OOM-restart, then runs
// the kill command and asserts it reports the OOMKilled service as a failure.
func assertOOMDetectedAtKill(t *testing.T, namespace, groupRef string) {
	t.Helper()

	pod := waitForPodInNamespace(t, namespace, 30*time.Second)
	require.NotNil(t, pod, "should find the service pod in the namespace")

	pod = waitForContainerRestart(t, namespace, pod.Name, 60*time.Second)
	require.True(t, hasContainerRestarted(pod, ""),
		"some container should have restarted after OOMKill")

	cfg, err := config.LoadConfigV2()
	require.NoError(t, err)

	err = commands.RunKillWithOptions(context.Background(), cfg, groupRef)
	assert.Error(t, err, "kill command should detect OOMKilled service as a failure")
}

func waitForPodInNamespace(t *testing.T, namespace string, timeout time.Duration) *corev1.Pod {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		podList, err := globalK8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "testkube.io/root=test-exec",
		})
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

func hasContainerRestarted(pod *corev1.Pod, containerName string) bool {
	for _, s := range pod.Status.ContainerStatuses {
		if containerName == "" || s.Name == containerName {
			if s.RestartCount > 0 {
				return true
			}
		}
	}
	return false
}

func waitForContainerRestart(t *testing.T, namespace, name string, timeout time.Duration) *corev1.Pod {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		pod, err := globalK8sClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		require.NoError(t, err)

		if hasContainerRestarted(pod, "") {
			return pod
		}

		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting for pod %s container to restart", name)
		case <-time.After(500 * time.Millisecond):
		}
	}
}
