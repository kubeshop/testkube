package toolkit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestServicePodLifecycle_Integration(t *testing.T) {
	test.IntegrationTest(t)

	tests := []struct {
		name         string
		command      string
		wantPhase    corev1.PodPhase
		wantRunning  bool
		wantExitCode *int32
	}{
		{
			name:      "failing pod is detected",
			command:   "exit 1",
			wantPhase: corev1.PodFailed,
		},
		{
			name:        "running pod has IP",
			command:     "sleep 300",
			wantRunning: true,
		},
		{
			name:         "exit code is preserved",
			command:      "exit 42",
			wantPhase:    corev1.PodFailed,
			wantExitCode: common.Ptr(int32(42)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace := createTestNamespace(t)
			t.Cleanup(func() { deleteTestNamespace(t, namespace) })

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: namespace,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:    "main",
						Image:   "busybox:1.36",
						Command: []string{"sh", "-c", tt.command},
					}},
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			_, err := globalK8sClient.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
			require.NoError(t, err)

			p := waitForPod(t, ctx, namespace, pod.Name, tt.wantRunning)

			if tt.wantPhase != "" {
				assert.Equal(t, tt.wantPhase, p.Status.Phase)
			}
			if tt.wantRunning {
				assert.Equal(t, corev1.PodRunning, p.Status.Phase)
				assert.NotEmpty(t, p.Status.PodIP, "running pod should have IP")
			}
			if tt.wantExitCode != nil {
				require.Len(t, p.Status.ContainerStatuses, 1)
				terminated := p.Status.ContainerStatuses[0].State.Terminated
				require.NotNil(t, terminated)
				assert.Equal(t, *tt.wantExitCode, terminated.ExitCode)
			}
		})
	}
}

func waitForPod(t *testing.T, ctx context.Context, namespace, name string, wantRunning bool) *corev1.Pod {
	for {
		p, err := globalK8sClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		require.NoError(t, err)

		if wantRunning && p.Status.Phase == corev1.PodRunning {
			return p
		}
		if !wantRunning && (p.Status.Phase == corev1.PodFailed || p.Status.Phase == corev1.PodSucceeded) {
			return p
		}

		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting for pod %s", name)
		case <-time.After(500 * time.Millisecond):
		}
	}
}
