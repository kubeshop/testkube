package agent

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

func apiServerPod(name, namespace string, ready bool, waitingReason string) *corev1.Pod {
	status := corev1.ConditionFalse
	if ready {
		status = corev1.ConditionTrue
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: map[string]string{"app.kubernetes.io/name": "api-server"}},
		Status: corev1.PodStatus{
			Phase:      corev1.PodRunning,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: status}},
		},
	}
	if waitingReason != "" {
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "api", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: waitingReason}}}}
	}
	return pod
}

func TestAPIServerValidator(t *testing.T) {
	tests := []struct {
		name        string
		pods        []*corev1.Pod
		connectErr  error
		wantValid   bool
		wantMessage string
		wantDetails string
		wantBreak   bool
	}{
		{name: "valid when every pod is ready", pods: []*corev1.Pod{apiServerPod("api-1", "testkube", true, ""), apiServerPod("api-2", "testkube", true, "")}, wantValid: true},
		{name: "invalid when there is no API server in the namespace", pods: []*corev1.Pod{apiServerPod("api-1", "other", true, "")}, wantMessage: ErrAPIServerNotFound.Message, wantDetails: "namespace testkube"},
		{name: "names the reason of a crashing pod", pods: []*corev1.Pod{apiServerPod("api-1", "testkube", true, ""), apiServerPod("api-2", "testkube", false, "CrashLoopBackOff")}, wantMessage: ErrAPIServerNotReady.Message, wantDetails: "api-2: container api is CrashLoopBackOff"},
		{name: "invalid when a pod is not ready", pods: []*corev1.Pod{apiServerPod("api-1", "testkube", false, "")}, wantMessage: ErrAPIServerNotReady.Message, wantDetails: "api-1: not ready"},
		{name: "stops the chain when the cluster is unreachable", connectErr: errors.New("no kubeconfig"), wantMessage: ErrClusterUnreachable.Message, wantDetails: "no kubeconfig", wantBreak: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			for _, pod := range tt.pods {
				_, err := client.CoreV1().Pods(pod.Namespace).Create(t.Context(), pod, metav1.CreateOptions{})
				require.NoError(t, err)
			}
			connect := func() (kubernetes.Interface, error) { return client, tt.connectErr }

			result := NewAPIServerValidator(connect, "testkube").Validate(nil)

			if tt.wantValid {
				assert.Equal(t, validators.StatusValid, result.Status)
				assert.Empty(t, result.Errors)
				assert.Equal(t, "2/2 pods ready in namespace testkube", result.AdditionalInfo)
				return
			}
			assert.Equal(t, validators.StatusInvalid, result.Status)
			require.Len(t, result.Errors, 1)
			assert.Equal(t, tt.wantMessage, result.Errors[0].Message)
			assert.Contains(t, result.Errors[0].Details, tt.wantDetails)
			assert.Equal(t, tt.wantBreak, result.BreakValidationChain)
		})
	}
}
