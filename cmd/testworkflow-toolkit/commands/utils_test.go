package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			err := Run(gitDiagnostics, "sh", "-c", tt.script)
			assert.Less(t, time.Since(start), 2*time.Second)

			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestRunWithRetry(t *testing.T) {
	tests := []struct {
		name         string
		stderr       string
		wantErr      string
		wantAttempts int
	}{
		{
			name:         "retries a failure that can be transient",
			stderr:       "fatal: unable to access 'https://github.com/org/repo.git/': Could not resolve host: github.com",
			wantAttempts: 3,
		},
		{
			name:         "does not retry a credential that the server refused",
			stderr:       "fatal: could not read Username for 'https://github.com': terminal prompts disabled",
			wantAttempts: 1,
		},
		{
			name:         "does not retry an SSH key that the server refused on a line before the fatal line",
			stderr:       "git@github.com: Permission denied (publickey).\nfatal: Could not read from remote repository.",
			wantErr:      "exit status 128: fatal: Could not read from remote repository.",
			wantAttempts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			counter := filepath.Join(dir, "attempts")
			stderr := filepath.Join(dir, "stderr")
			require.NoError(t, os.WriteFile(stderr, []byte(tt.stderr+"\n"), 0o600))
			script := fmt.Sprintf("echo x >> %s; cat %s >&2; exit 128", counter, stderr)

			err := RunWithRetry(3, time.Millisecond, isGitAuthError, gitDiagnostics, "sh", "-c", script)

			wantErr := tt.wantErr
			if wantErr == "" {
				wantErr = "exit status 128: " + tt.stderr
			}
			assert.EqualError(t, err, wantErr)
			content, readErr := os.ReadFile(counter)
			require.NoError(t, readErr)
			assert.Equal(t, tt.wantAttempts, strings.Count(string(content), "x"))
		})
	}
}
