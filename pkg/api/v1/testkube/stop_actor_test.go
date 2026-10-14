package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStopActor_Sentence(t *testing.T) {
	tests := []struct {
		name  string
		actor StopActor
		want  string
	}{
		{name: "a user is a person", actor: StopActorUser, want: "by the user"},
		{name: "the control plane is not a person", actor: StopActorControlPlane, want: "by the control plane"},
		{name: "trigger", actor: StopActorTrigger, want: "because its trigger was deleted"},
		{name: "fail-fast", actor: StopActorFailFast, want: "because another parallel worker failed"},
		{name: "runner", actor: StopActorRunner, want: "by the runner"},
		{name: "quality loop", actor: StopActorQualityLoop, want: "by the quality loop"},
		{name: "the abort endpoint acts for a person", actor: StopActorAPI, want: "through the API"},
		{name: "system", actor: StopActorSystem, want: "by the system"},
		{name: "unknown value falls back to the system", actor: StopActor("later-added"), want: "by the system"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.actor.Sentence())
		})
	}
}

func TestStopActor_IsPerson(t *testing.T) {
	tests := []struct {
		name  string
		actor StopActor
		want  bool
	}{
		{name: "a user is a person", actor: StopActorUser, want: true},
		{name: "the abort endpoint acts for a person", actor: StopActorAPI, want: true},
		{name: "the control plane is not a person", actor: StopActorControlPlane, want: false},
		{name: "a stop without an actor has no person", actor: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.actor.IsPerson())
		})
	}
}
