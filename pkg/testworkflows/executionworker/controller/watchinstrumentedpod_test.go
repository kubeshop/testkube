package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/stretchr/testify/assert"
)

func TestMapPodEventRef(t *testing.T) {
	t.Run("emits pod-level events", func(t *testing.T) {
		ev := &corev1.Event{}
		ref, emit := mapPodEventRef("1", "step/a", ev)
		assert.True(t, emit)
		assert.Equal(t, "", ref)
	})

	t.Run("emits non-start lifecycle container events for current container", func(t *testing.T) {
		ev := &corev1.Event{Reason: "Killing", InvolvedObject: corev1.ObjectReference{FieldPath: "spec.containers{1}"}}
		ref, emit := mapPodEventRef("1", "step/a", ev)
		assert.True(t, emit)
		assert.Equal(t, "step/a", ref)
	})

	t.Run("skips created and started events for current container", func(t *testing.T) {
		created := &corev1.Event{Reason: "Created", InvolvedObject: corev1.ObjectReference{FieldPath: "spec.containers{1}"}}
		started := &corev1.Event{Reason: "Started", InvolvedObject: corev1.ObjectReference{FieldPath: "spec.containers{1}"}}

		_, emitCreated := mapPodEventRef("1", "step/a", created)
		_, emitStarted := mapPodEventRef("1", "step/a", started)

		assert.False(t, emitCreated)
		assert.False(t, emitStarted)
	})

	t.Run("skips events from another container", func(t *testing.T) {
		ev := &corev1.Event{Reason: "Killing", InvolvedObject: corev1.ObjectReference{FieldPath: "spec.containers{2}", UID: types.UID("x")}}
		_, emit := mapPodEventRef("1", "step/a", ev)
		assert.False(t, emit)
	})
}
