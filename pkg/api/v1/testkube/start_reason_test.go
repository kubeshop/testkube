package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartReason_Sentence(t *testing.T) {
	tests := []struct {
		name   string
		reason StartReason
		want   string
	}{
		{name: "image pull", reason: StartReasonImagePullFailed, want: "the image could not be pulled"},
		{name: "definition", reason: StartReasonDefinitionInvalid, want: "the workflow definition is invalid"},
		{name: "resource", reason: StartReasonResourceFailed, want: "a secret, config map, or volume claim could not be created"},
		{name: "job", reason: StartReasonJobCreateFailed, want: "the job could not be created"},
		{name: "unknown phase", reason: StartReasonUnknown, want: "the runner could not start the execution"},
		{name: "token from a newer runner has no words yet", reason: StartReason("later-added"), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.reason.Sentence())
		})
	}
}
