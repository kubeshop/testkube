package watchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestExecutionState_CurrentCause(t *testing.T) {
	waiting := func(name, reason, message string) corev1.ContainerStatus {
		return corev1.ContainerStatus{Name: name, State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason, Message: message}}}
	}
	unschedulable := corev1.PodCondition{
		Type:    corev1.PodScheduled,
		Status:  corev1.ConditionFalse,
		Reason:  corev1.PodReasonUnschedulable,
		Message: "0/1 nodes are available: 1 Insufficient cpu.",
	}

	tests := []struct {
		name string
		pod  *corev1.Pod
		want *testkube.Cause
	}{
		{
			name: "returns nil without a pod",
			want: nil,
		},
		{
			name: "returns unschedulable with the scheduler message",
			pod:  &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, Conditions: []corev1.PodCondition{unschedulable}}},
			want: &testkube.Cause{Reason: "unschedulable", Message: "0/1 nodes are available: 1 Insufficient cpu."},
		},
		{
			name: "returns image-pull-failed for an init container",
			pod: &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, InitContainerStatuses: []corev1.ContainerStatus{
				waiting("tktw-init", "ErrImagePull", "pull access denied"),
			}}},
			want: &testkube.Cause{Reason: "image-pull-failed", Message: "pull access denied"},
		},
		{
			name: "returns config-missing for a container that waits for a secret",
			pod: &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, ContainerStatuses: []corev1.ContainerStatus{
				waiting("1", "CreateContainerConfigError", `secret "does-not-exist" not found`),
			}}},
			want: &testkube.Cause{Reason: "config-missing", Message: `secret "does-not-exist" not found`},
		},
		{
			name: "ignores a container that waits for its normal start",
			pod: &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, ContainerStatuses: []corev1.ContainerStatus{
				waiting("1", "ContainerCreating", ""),
			}}},
			want: nil,
		},
		{
			name: "ignores a container that is not a step container",
			pod: &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, ContainerStatuses: []corev1.ContainerStatus{
				waiting("sidecar", "ImagePullBackOff", "Back-off pulling image"),
			}}},
			want: nil,
		},
		{
			name: "returns nil for a finished pod",
			pod:  &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed, Conditions: []corev1.PodCondition{unschedulable}}},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pod Pod
			if tt.pod != nil {
				pod = NewPod(tt.pod)
			}
			state := NewExecutionState(nil, pod, NewJobEvents(nil), NewPodEvents(nil), nil)

			assert.Equal(t, tt.want, state.CurrentCause())
		})
	}
}
