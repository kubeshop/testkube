package orchestration

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
)

func TestExecution_Run(t *testing.T) {
	tests := []struct {
		name              string
		script            string
		abortWhileRunning bool
		want              executionResult
	}{
		{
			name:   "reports no details for a process that exits",
			script: "exit 3",
			want:   executionResult{ExitCode: 3},
		},
		{
			name:   "reports the process-killed message for a process that a signal from outside killed",
			script: "kill -9 $$",
			want: executionResult{
				Aborted:  true,
				ExitCode: constants.CodeAborted,
				Details:  "the test process was killed, possibly by an out-of-memory kill (signal: killed)",
			},
		},
		{
			name:   "reports no details for a process that SIGTERM stopped, because a stop of the pod has its own cause",
			script: "kill -TERM $$",
			want:   executionResult{Aborted: true, ExitCode: constants.CodeAborted},
		},
		{
			name:   "reports no details for a process that crashed with another signal",
			script: "kill -SEGV $$",
			want:   executionResult{Aborted: true, ExitCode: constants.CodeAborted},
		},
		{
			name:              "reports no details when the init process aborts the group while the process runs",
			script:            "sleep 30",
			abortWhileRunning: true,
			want:              executionResult{Aborted: true, ExitCode: constants.CodeAborted},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := newExecutionGroup(io.Discard, io.Discard)
			execution := group.CreateWithContext(context.Background(), "/bin/sh", []string{"-c", tt.script})
			if tt.abortWhileRunning {
				go abortWhenStarted(group, execution)
			}

			result, err := execution.Run()

			require.NoError(t, err)
			assert.Equal(t, tt.want, *result)
		})
	}
}

// abortWhenStarted aborts the group after the process starts. An abort before the start takes the
// early return of Run, and that return does not test the branch after the process exits.
// It stops to wait after a time limit, so the goroutine ends also when the process does not start.
func abortWhenStarted(group *executionGroup, execution *execution) {
	for limit := time.Now().Add(5 * time.Second); time.Now().Before(limit); {
		execution.cmdMu.Lock()
		started := execution.cmd != nil && execution.cmd.Process != nil
		execution.cmdMu.Unlock()
		if started {
			group.Abort()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
