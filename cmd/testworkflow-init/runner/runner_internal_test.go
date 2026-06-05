package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
)

func TestSetupAbortHandlerStopsCleanly(t *testing.T) {
	orchestration.Executions.ClearAbortedStatus()
	t.Cleanup(func() {
		orchestration.Executions.ClearAbortedStatus()
	})

	ctx, cancel := context.WithCancel(context.Background())
	stop := setupAbortHandler(ctx)
	stop()
	cancel()

	time.Sleep(20 * time.Millisecond)
	assert.False(t, orchestration.Executions.IsAborted())
}

func TestSetupAbortHandlerAbortsOnContextCancel(t *testing.T) {
	orchestration.Executions.ClearAbortedStatus()
	t.Cleanup(func() {
		orchestration.Executions.ClearAbortedStatus()
	})

	ctx, cancel := context.WithCancel(context.Background())
	stop := setupAbortHandler(ctx)
	defer stop()

	cancel()

	require.Eventually(t, orchestration.Executions.IsAborted, time.Second, 10*time.Millisecond)
}
