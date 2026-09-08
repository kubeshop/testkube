package executionworkertypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDestroyOptions_TerminationActor(t *testing.T) {
	tests := []struct {
		name         string
		options      DestroyOptions
		defaultActor AbortActor
		want         AbortActor
	}{
		{
			name:         "actor set",
			options:      DestroyOptions{Actor: AbortActorTrigger},
			defaultActor: AbortActorSystem,
			want:         AbortActorTrigger,
		},
		{
			name:         "no actor uses the default",
			options:      DestroyOptions{},
			defaultActor: AbortActorUser,
			want:         AbortActorUser,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.options.TerminationActor(tt.defaultActor))
		})
	}
}

func TestAbortActor_Sentence(t *testing.T) {
	tests := []struct {
		name  string
		actor AbortActor
		want  string
	}{
		{name: "user", actor: AbortActorUser, want: "by the user"},
		{name: "control plane", actor: AbortActorControlPlane, want: "by the control plane"},
		{name: "trigger", actor: AbortActorTrigger, want: "because its trigger was deleted"},
		{name: "fail-fast", actor: AbortActorFailFast, want: "because another parallel worker failed"},
		{name: "runner", actor: AbortActorRunner, want: "by the runner"},
		{name: "api", actor: AbortActorAPI, want: "through the API"},
		{name: "system", actor: AbortActorSystem, want: "by the system"},
		{name: "unknown value falls back to the system", actor: AbortActor("later-added"), want: "by the system"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.actor.Sentence())
		})
	}
}
