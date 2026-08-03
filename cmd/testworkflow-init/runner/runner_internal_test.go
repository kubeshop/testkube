package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
)

func TestSetupAbortHandler(t *testing.T) {
	orchestration.Initialize()

	tests := []struct {
		name        string
		detach      bool
		wantAborted bool
	}{
		{
			name:        "aborts executions when the context ends during the run",
			detach:      false,
			wantAborted: true,
		},
		{
			name:        "never aborts once the handler is detached",
			detach:      true,
			wantAborted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orchestration.Executions.ClearAbortedStatus()
			t.Cleanup(orchestration.Executions.ClearAbortedStatus)

			ctx, cancel := context.WithCancel(context.Background())
			stop := setupAbortHandler(ctx)
			if tt.detach {
				assert.True(t, stop(), "handler should detach before the context ends")
			} else {
				defer stop()
			}
			cancel()

			if tt.wantAborted {
				require.Eventually(t, orchestration.Executions.IsAborted, time.Second, 5*time.Millisecond)
			} else {
				time.Sleep(50 * time.Millisecond)
				assert.False(t, orchestration.Executions.IsAborted(),
					"a detached handler must not abort a subsequent run")
			}
		})
	}
}
