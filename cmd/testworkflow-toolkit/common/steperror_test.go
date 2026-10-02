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

			writeStepError("fatal: repository 'https://github.com/org/absent.git/' not found")

			content, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, "fatal: repository 'https://github.com/org/absent.git/' not found", string(content))
		})
	}
}

func TestWriteStepReason(t *testing.T) {
	tests := []struct {
		name     string
		setPath  bool
		reason   testkube.StopReason
		wantFile bool
	}{
		{
			name:     "writes the code to the file that TK_REASON_FILE names",
			setPath:  true,
			reason:   testkube.StopReasonGitAuthFailed,
			wantFile: true,
		},
		{
			name:   "an init process of an earlier release names no file, so the toolkit writes none",
			reason: testkube.StopReasonGitAuthFailed,
		},
		{
			name:    "a failure without a code writes no file",
			setPath: true,
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
			if !tt.wantFile {
				assert.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, string(tt.reason), string(content))
		})
	}
}
