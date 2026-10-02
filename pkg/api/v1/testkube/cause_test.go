package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCause_String(t *testing.T) {
	tests := []struct {
		name  string
		cause Cause
		want  string
	}{
		{
			name:  "renders a start reason with the reported text",
			cause: Cause{Reason: string(StartReasonImagePullFailed), Message: "Back-off pulling image \"missing:tag\""},
			want:  "the image could not be pulled: Back-off pulling image \"missing:tag\"",
		},
		{
			name:  "renders only the words when Kubernetes reported no text",
			cause: Cause{Reason: string(StopReasonConfigMissing)},
			want:  "a secret or a config map that a container needs is not available",
		},
		{
			name:  "keeps a code that has no words",
			cause: Cause{Reason: "later-added", Message: "details"},
			want:  "later-added: details",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cause.String())
		})
	}
}
