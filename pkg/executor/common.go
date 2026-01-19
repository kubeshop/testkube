package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// IsPodFailed checks if pod failed
// pod can be in wait state with reason which is error for us on the end
func IsPodFailed(pod *corev1.Pod) (err error) {
	if pod.Status.Phase == corev1.PodFailed {
		return errors.New(pod.Status.Message)
	}

	for _, initContainerStatus := range pod.Status.InitContainerStatuses {
		waitState := initContainerStatus.State.Waiting
		// TODO there could be more edge cases but didn't found any constants in go libraries
		if waitState != nil && IsWaitStateFailed(waitState.Reason) {
			return errors.New(waitState.Message)
		}
	}

	return
}

// IsWaitStateFailed defines possible failed wait state
// those states are defined and throwed as errors in Kubernetes runtime
// https://github.com/kubernetes/kubernetes/blob/127f33f63d118d8d61bebaba2a240c60f71c824a/pkg/kubelet/kuberuntime/kuberuntime_container.go#L59
func IsWaitStateFailed(state string) bool {
	var failedWaitingStates = []string{
		"CreateContainerConfigError",
		"PreCreateHookError",
		"CreateContainerError",
		"PreStartHookError",
		"PostStartHookError",
	}

	for _, fws := range failedWaitingStates {
		if state == fws {
			return true
		}
	}

	return false
}

// GetPodByName returns job pod by name
func GetPodByName(ctx context.Context, podsClient tcorev1.PodInterface, podName string, retryNr, retryCount int) (*corev1.Pod, error) {
	pod, err := podsClient.Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if retryNr == retryCount {
		return nil, fmt.Errorf("retry count exceeeded, there is no active pod with given name=%s", podName)
	}
	if pod == nil {
		time.Sleep(time.Duration(retryNr * 500 * int(time.Millisecond))) // increase backoff timeout
		return GetPodByName(ctx, podsClient, podName, retryNr+1, retryCount)
	}
	return pod, nil
}
