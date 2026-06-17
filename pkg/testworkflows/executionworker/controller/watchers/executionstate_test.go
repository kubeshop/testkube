package watchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodExecutionError_UsesPodEventsForGenericEvictionCondition(t *testing.T) {
	state := NewExecutionState(
		nil,
		NewPod(&corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{
			Type:    corev1.DisruptionTarget,
			Status:  corev1.ConditionTrue,
			Reason:  "EvictionByEvictionAPI",
			Message: "Eviction API: evicting",
		}}}}),
		NewJobEvents(nil),
		NewPodEvents([]*corev1.Event{{
			Reason:  "Evicted",
			Message: "The node was low on resource: ephemeral-storage",
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.Now(),
			},
		}}),
		nil,
	)

	assert.Equal(
		t,
		"EvictionByEvictionAPI: Eviction API: evicting (pod event: Evicted: The node was low on resource: ephemeral-storage)",
		state.PodExecutionError(),
	)
}

func TestPodExecutionError_UsesPodEventsWhenPodErrorEmpty(t *testing.T) {
	state := NewExecutionState(
		nil,
		NewPod(&corev1.Pod{}),
		NewJobEvents(nil),
		NewPodEvents([]*corev1.Event{{
			Reason:  "Evicted",
			Message: "The node was low on resource: memory",
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.Now(),
			},
		}}),
		nil,
	)

	assert.Equal(t, "Evicted: The node was low on resource: memory", state.PodExecutionError())
}
