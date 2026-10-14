package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	waitDelay := runWaitDelay
	runWaitDelay = 100 * time.Millisecond
	t.Cleanup(func() { runWaitDelay = waitDelay })

	tests := []struct {
		name    string
		script  string
		wantErr string
	}{
		{
			name:    "returns no error when the command passes",
			script:  "echo 'warning: ignored' >&2",
			wantErr: "",
		},
		{
			name:    "adds the fatal line when hints follow it",
			script:  "echo 'ERROR: Repository not found.' >&2; echo 'fatal: Could not read from remote repository.' >&2; echo >&2; echo 'and the repository exists.' >&2; exit 128",
			wantErr: "exit status 128: fatal: Could not read from remote repository.",
		},
		{
			name:    "adds the last line that is not empty when no line is fatal",
			script:  "printf 'Receiving objects: 50%%\\rerror: pathspec did not match\\n\\n' >&2; exit 1",
			wantErr: "exit status 1: error: pathspec did not match",
		},
		{
			name:    "adds a last line without a line end",
			script:  "printf 'fatal: bad revision' >&2; exit 128",
			wantErr: "exit status 128: fatal: bad revision",
		},
		{
			name:    "returns the exit status when the standard error is empty",
			script:  "echo 'fatal: only on the standard output'; exit 2",
			wantErr: "exit status 2",
		},
		{
			name:    "returns no error when a child process keeps the standard error open",
			script:  "sleep 3 >&2 &",
			wantErr: "",
		},
		{
			name:    "adds the fatal line when a child process keeps the standard error open",
			script:  "echo 'fatal: unable to access' >&2; sleep 3 >&2 & exit 128",
			wantErr: "exit status 128: fatal: unable to access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			err := Run("sh", "-c", tt.script)
			assert.Less(t, time.Since(start), 2*time.Second)

			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
