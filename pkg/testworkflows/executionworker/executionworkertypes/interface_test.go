package executionworkertypes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestDestroyOptions_TerminationActor(t *testing.T) {
	tests := []struct {
		name         string
		options      DestroyOptions
		defaultActor testkube.StopActor
		want         testkube.StopActor
	}{
		{
			name:         "actor set",
			options:      DestroyOptions{Actor: testkube.StopActorTrigger},
			defaultActor: testkube.StopActorSystem,
			want:         testkube.StopActorTrigger,
		},
		{
			name:         "no actor uses the default",
			options:      DestroyOptions{},
			defaultActor: testkube.StopActorUser,
			want:         testkube.StopActorUser,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.options.TerminationActor(tt.defaultActor))
		})
	}
}
