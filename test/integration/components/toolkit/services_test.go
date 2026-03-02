package toolkit_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestServiceEnvOverride_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	// Service overrides MY_VAR and uses expression that must resolve to service's value
	services := map[string]testworkflowsv1.ServiceSpec{
		"env-test": {
			IndependentServiceSpec: testworkflowsv1.IndependentServiceSpec{
				StepRun: testworkflowsv1.StepRun{
					ContainerConfig: testworkflowsv1.ContainerConfig{
						Image: "busybox:1.36",
						Env: []testworkflowsv1.EnvVar{
							{EnvVar: corev1.EnvVar{Name: "MY_VAR", Value: "service-value"}},
						},
					},
					Shell: common.Ptr(`[ "{{ env.MY_VAR }}" = "service-value" ]`),
				},
				Pod: &testworkflowsv1.PodConfig{},
			},
		},
	}

	observer, err := startPodObserver(t, namespace)
	require.NoError(t, err)
	t.Cleanup(observer.Stop)

	err = executeServices(t, services, "test-group")
	require.NoError(t, err, "service should start successfully")

	err = observer.WaitForPods(1, 30*time.Second)
	require.NoError(t, err, "should observe 1 service pod")

	pods := observer.GetCreatedPods()
	require.Len(t, pods, 1)

	// Wait for pod to complete and verify success
	pod := waitForPodTermination(t, namespace, pods[0].Name, 30*time.Second)
	assert.Equal(t, corev1.PodSucceeded, pod.Status.Phase,
		"service pod should succeed - env expression resolved to service's value")
}

// TestServiceFailureDetection_Integration verifies that failing services are detected.
func TestServiceFailureDetection_Integration(t *testing.T) {
	test.IntegrationTest(t)

	namespace := createTestNamespace(t)
	t.Cleanup(func() { deleteTestNamespace(t, namespace) })

	_, _, cleanupCP := setupTestWithControlPlane(t, namespace)
	t.Cleanup(cleanupCP)

	services := map[string]testworkflowsv1.ServiceSpec{
		"failing-service": {
			IndependentServiceSpec: testworkflowsv1.IndependentServiceSpec{
				StepRun: testworkflowsv1.StepRun{
					ContainerConfig: testworkflowsv1.ContainerConfig{
						Image: "busybox:1.36",
					},
					Shell: common.Ptr(`exit 1`),
				},
				Pod: &testworkflowsv1.PodConfig{},
			},
		},
	}

	err := executeServices(t, services, "test-group")
	assert.Error(t, err, "should detect service failure")
}

func executeServices(t *testing.T, services map[string]testworkflowsv1.ServiceSpec, groupRef string) error {
	t.Helper()

	data, err := json.Marshal(services)
	require.NoError(t, err)
	encoded := base64.StdEncoding.EncodeToString(data)

	cfg, err := config.LoadConfigV2()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- commands.RunServicesWithOptions(encoded, cfg, true, groupRef)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func waitForPodTermination(t *testing.T, namespace, name string, timeout time.Duration) *corev1.Pod {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		pod, err := globalK8sClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		require.NoError(t, err)

		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			return pod
		}

		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting for pod %s to terminate", name)
		case <-time.After(500 * time.Millisecond):
		}
	}
}
