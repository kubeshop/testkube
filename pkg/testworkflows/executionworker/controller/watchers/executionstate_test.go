package watchers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	warning := func(reason, message string) *corev1.Event {
		return &corev1.Event{Type: corev1.EventTypeWarning, Reason: reason, Message: message}
	}
	normal := func(reason string) *corev1.Event {
		return &corev1.Event{Type: corev1.EventTypeNormal, Reason: reason}
	}
	at := func(event *corev1.Event, second int) *corev1.Event {
		event.LastTimestamp = metav1.Time{Time: time.Date(2026, 1, 1, 10, 0, second, 0, time.UTC)}
		return event
	}
	pending := &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}

	tests := []struct {
		name      string
		pod       *corev1.Pod
		podEvents []*corev1.Event
		jobEvents []*corev1.Event
		want      *testkube.Cause
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
			name:      "returns volume-mount-failed from the latest mount warning",
			pod:       pending,
			podEvents: []*corev1.Event{normal("Scheduled"), warning("FailedMount", `secret "missing" not found`)},
			want:      &testkube.Cause{Reason: "volume-mount-failed", Message: `secret "missing" not found`},
		},
		{
			name:      "ignores a scheduling warning that a later progress event passed",
			pod:       pending,
			podEvents: []*corev1.Event{warning("FailedScheduling", "0/1 nodes are available"), normal("Scheduled")},
			want:      nil,
		},
		{
			name:      "keeps a newer failure when an older progress event comes later in the list",
			pod:       pending,
			podEvents: []*corev1.Event{at(warning("FailedScheduling", "0/1 nodes are available"), 20), at(normal("Scheduled"), 10)},
			want:      &testkube.Cause{Reason: "unschedulable", Message: "0/1 nodes are available"},
		},
		{
			name:      "ignores an older failure when a newer progress event comes earlier in the list",
			pod:       pending,
			podEvents: []*corev1.Event{at(normal("Scheduled"), 20), at(warning("FailedScheduling", "0/1 nodes are available"), 10)},
			want:      nil,
		},
		{
			name:      "ignores an event with a cause reason that is not a warning",
			pod:       pending,
			podEvents: []*corev1.Event{normal("FailedMount")},
			want:      nil,
		},
		{
			name:      "returns admission-denied from the job before the pod exists",
			jobEvents: []*corev1.Event{warning("FailedCreate", `admission webhook "policy" denied the request: privileged containers are not allowed`)},
			want:      &testkube.Cause{Reason: "admission-denied", Message: `admission webhook "policy" denied the request: privileged containers are not allowed`},
		},
		{
			name:      "returns nil when the job created the pod after an earlier create warning, before the pod shows",
			jobEvents: []*corev1.Event{at(warning("FailedCreate", "denied"), 1), at(normal("SuccessfulCreate"), 2)},
			want:      nil,
		},
		{
			name:      "ignores a job create warning when the pod exists",
			pod:       pending,
			jobEvents: []*corev1.Event{warning("FailedCreate", "denied")},
			want:      nil,
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
			state := NewExecutionState(nil, pod, NewJobEvents(tt.jobEvents), NewPodEvents(tt.podEvents), nil)

			assert.Equal(t, tt.want, state.CurrentCause())
		})
	}
}
