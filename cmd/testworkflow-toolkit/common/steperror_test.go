package common

import (
	"errors"
	"fmt"
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

// codeError is an error that knows its code, as a toolkit command builds one.
type codeError struct{ reason testkube.StopReason }

func (e codeError) Error() string               { return "the step failed" }
func (e codeError) Reason() testkube.StopReason { return e.reason }

func TestReasonOf(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want testkube.StopReason
	}{
		{
			name: "an error that knows its code gives the code",
			err:  codeError{reason: testkube.StopReasonGitAuthFailed},
			want: testkube.StopReasonGitAuthFailed,
		},
		{
			name: "a wrapped error that knows its code gives the code",
			err:  fmt.Errorf("clone: %w", codeError{reason: testkube.StopReasonServiceNotReady}),
			want: testkube.StopReasonServiceNotReady,
		},
		{
			name: "a plain error gives no code",
			err:  errors.New("the step failed"),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ReasonOf(tt.err))
		})
	}
}

func TestWithReason(t *testing.T) {
	cause := errors.New("could not create cloud client: no token")
	tests := []struct {
		name        string
		err         error
		wantNil     bool
		wantMessage string
		wantReason  testkube.StopReason
	}{
		{
			name:    "a nil error stays nil, so the step does not fail",
			err:     WithReason(testkube.StopReasonArtifactUploadFailed, nil),
			wantNil: true,
		},
		{
			name:        "an error with a code keeps its message and gives the code",
			err:         WithReason(testkube.StopReasonArtifactUploadFailed, cause),
			wantMessage: "could not create cloud client: no token",
			wantReason:  testkube.StopReasonArtifactUploadFailed,
		},
		{
			name:        "a wrapped error with a code still gives the code",
			err:         fmt.Errorf("artifacts: %w", WithReason(testkube.StopReasonArtifactUploadFailed, cause)),
			wantMessage: "artifacts: could not create cloud client: no token",
			wantReason:  testkube.StopReasonArtifactUploadFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantNil {
				assert.NoError(t, tt.err)
				return
			}
			assert.EqualError(t, tt.err, tt.wantMessage)
			assert.Equal(t, tt.wantReason, ReasonOf(tt.err))
			assert.ErrorIs(t, tt.err, cause)
		})
	}
}
