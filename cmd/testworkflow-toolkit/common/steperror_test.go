package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestWriteStepError(t *testing.T) {
	tests := []struct {
		name     string
		existing *string
	}{
		{
			name: "writes the message to the file that TK_ERR_FILE names",
		},
		{
			name:     "replaces a longer message that an earlier write left in the file",
			existing: common.Ptr("cannot fetch the artifacts of an earlier attempt: connection refused"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "error")
			if tt.existing != nil {
				require.NoError(t, os.WriteFile(path, []byte(*tt.existing), 0666))
			}
			t.Setenv(constants.EnvStepErrorFile, path)

			writeStepError("error cloning repository: exit status 128")

			content, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, "error cloning repository: exit status 128", string(content))
		})
	}
}

func TestWriteStepReason(t *testing.T) {
	tests := []struct {
		name    string
		setPath bool
		reason  testkube.StopReason
		want    string
		wantAny bool
	}{
		{
			name:    "writes the code to the file that TK_REASON_FILE names",
			setPath: true,
			reason:  testkube.StopReasonGitAuthFailed,
			want:    "git-auth-failed",
			wantAny: true,
		},
		{
			// An init image from an earlier release names no reason file, so the toolkit writes
			// none and the step keeps its message alone.
			name:   "writes no file when the init process names none",
			reason: testkube.StopReasonGitAuthFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "reason")
			if tt.setPath {
				t.Setenv(constants.EnvStepReasonFile, path)
			} else {
				t.Setenv(constants.EnvStepReasonFile, "")
			}

			writeStepReason(string(tt.reason))

			content, err := os.ReadFile(path)
			if !tt.wantAny {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(content))
		})
	}
}
