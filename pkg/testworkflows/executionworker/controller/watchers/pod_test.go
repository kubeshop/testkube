package watchers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPod_FinishTimestamp(t *testing.T) {
	first := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	last := first.Add(5 * time.Second)
	containers := func(names ...string) []corev1.Container {
		result := make([]corev1.Container, len(names))
		for i, name := range names {
			result[i] = corev1.Container{Name: name}
		}
		return result
	}
	terminated := func(name string, finishedAt time.Time) corev1.ContainerStatus {
		return corev1.ContainerStatus{Name: name, State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Completed", FinishedAt: metav1.Time{Time: finishedAt}}}}
	}
	running := func(name string) corev1.ContainerStatus {
		return corev1.ContainerStatus{Name: name, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}
	}

	tests := []struct {
		name       string
		phase      corev1.PodPhase
		conditions []corev1.PodCondition
		spec       []corev1.Container
		statuses   []corev1.ContainerStatus
		want       time.Time
	}{
		{
			name:     "completes when every step container terminated and a sidecar still runs",
			spec:     containers("1", "2", "sidecar"),
			statuses: []corev1.ContainerStatus{terminated("1", first), terminated("2", last), running("sidecar")},
			want:     last,
		},
		{
			name:     "does not complete while a step container runs",
			spec:     containers("1", "2", "sidecar"),
			statuses: []corev1.ContainerStatus{terminated("1", first), running("2"), running("sidecar")},
			want:     time.Time{},
		},
		{
			name:       "completes a pod that succeeded at the time of its last condition",
			phase:      corev1.PodSucceeded,
			conditions: []corev1.PodCondition{{Type: corev1.PodReady, LastTransitionTime: metav1.Time{Time: last}}},
			spec:       containers("1"),
			statuses:   []corev1.ContainerStatus{terminated("1", first)},
			want:       last,
		},
		{
			name:     "does not complete a pod without step containers",
			spec:     containers("sidecar"),
			statuses: []corev1.ContainerStatus{terminated("sidecar", first)},
			want:     time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase := tt.phase
			if phase == "" {
				phase = corev1.PodRunning
			}
			pod := NewPod(&corev1.Pod{
				Spec:   corev1.PodSpec{Containers: tt.spec},
				Status: corev1.PodStatus{Phase: phase, Conditions: tt.conditions, ContainerStatuses: tt.statuses},
			})

			assert.Equal(t, tt.want, pod.FinishTimestamp())
			assert.Equal(t, !tt.want.IsZero(), pod.Finished())
		})
	}
}

func TestPod_Unschedulable(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "a message with two final periods ends with one",
			message: "0/2 nodes are available: 2 node(s) didn't match Pod's node affinity/selector. preemption: 0/2 nodes are available: 2 Preemption is not helpful for scheduling..",
			want:    "0/2 nodes are available: 2 node(s) didn't match Pod's node affinity/selector. preemption: 0/2 nodes are available: 2 Preemption is not helpful for scheduling.",
		},
		{
			name:    "a message with one final period stays",
			message: "0/1 nodes are available: 1 Insufficient cpu.",
			want:    "0/1 nodes are available: 1 Insufficient cpu.",
		},
		{
			name:    "a message without a final period stays",
			message: "0/1 nodes are available",
			want:    "0/1 nodes are available",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPod(&corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{
				Type:    corev1.PodScheduled,
				Status:  corev1.ConditionFalse,
				Reason:  corev1.PodReasonUnschedulable,
				Message: tt.message,
			}}}})
			got, ok := p.Unschedulable()
			assert.True(t, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
