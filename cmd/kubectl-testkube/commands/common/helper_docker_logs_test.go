//go:build unix

package common

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStreamDockerLogs guards the result of "testkube docker init" log streaming: the
// installation markers decide success or failure, docker errors surface as CLI errors,
// and a still-running "docker logs --follow" is stopped before the function returns.
// testdata/docker stands in for the docker CLI; the container name picks its scenario.
func TestStreamDockerLogs(t *testing.T) {
	testdata, err := filepath.Abs("testdata")
	require.NoError(t, err)
	// The fake docker needs the system PATH for sleep.
	fakeDockerPath := testdata + string(os.PathListSeparator) + os.Getenv("PATH")

	tests := []struct {
		name      string
		container string
		path      string // PATH for the docker lookup
		wantCode  ErrorCode
		wantErr   string
		// keepsRunning marks scenarios where docker logs would follow forever after the marker.
		keepsRunning bool
	}{
		{
			name:         "success marker returns no error",
			container:    "succeeded",
			path:         fakeDockerPath,
			keepsRunning: true,
		},
		{
			name:         "failure marker on stderr returns the failing line",
			container:    "failed",
			path:         fakeDockerPath,
			wantCode:     TKErrDockerInstallationFailed,
			wantErr:      "Testkube installation failed: chart not found",
			keepsRunning: true,
		},
		{
			name:      "docker logs exiting with an error",
			container: "missing",
			path:      fakeDockerPath,
			wantCode:  TKErrDockerLogStreamingFailed,
			wantErr:   "exit status 1",
		},
		{
			name:      "docker CLI not installed",
			container: "succeeded",
			path:      t.TempDir(), // empty, so no real docker is found either
			wantCode:  TKErrDockerLogStreamingFailed,
			wantErr:   `exec: "docker": executable file not found in $PATH`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pidFile := filepath.Join(t.TempDir(), "pid")
			t.Setenv("PATH", tt.path)
			t.Setenv("FAKE_DOCKER_PID_FILE", pidFile)

			cliErr := StreamDockerLogs(tt.container)

			if tt.wantCode == "" {
				assert.Nil(t, cliErr, "StreamDockerLogs(%q)", tt.container)
			} else {
				require.NotNil(t, cliErr, "StreamDockerLogs(%q): got nil, want %s", tt.container, tt.wantCode)
				assert.Equal(t, tt.wantCode, cliErr.Code, "StreamDockerLogs(%q) code", tt.container)
				assert.EqualError(t, cliErr.ActualError, tt.wantErr, "StreamDockerLogs(%q) error", tt.container)
			}

			if tt.keepsRunning {
				raw, err := os.ReadFile(pidFile)
				require.NoError(t, err, "fake docker did not start")
				pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
				require.NoError(t, err)
				// Signal 0 only checks existence: ESRCH means the process was stopped and reaped.
				assert.ErrorIs(t, syscall.Kill(pid, 0), syscall.ESRCH, "docker logs (pid %d) still running after StreamDockerLogs returned", pid)
			}
		})
	}
}
