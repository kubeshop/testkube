package testsuite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/test/fixtures"
)

func testGetRunning(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	running := fixtures.NewExecution("running-test",
		fixtures.WithStatus(testkube.RUNNING_TestWorkflowStatus),
	)
	err := repo.Insert(ctx, running)
	require.NoError(t, err)

	queued := fixtures.NewQueuedExecution("queued-for-running-test")
	err = repo.Insert(ctx, queued)
	require.NoError(t, err)

	results, err := repo.GetRunning(ctx)
	require.NoError(t, err)

	found := false
	for _, r := range results {
		if r.Id == running.Id {
			found = true
			break
		}
	}
	assert.True(t, found, "running execution should be found in GetRunning results")
}

func testGetFinished(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	passed := fixtures.NewPassedExecution("finished-passed")
	err := repo.Insert(ctx, passed)
	require.NoError(t, err)

	failed := fixtures.NewFailedExecution("finished-failed")
	err = repo.Insert(ctx, failed)
	require.NoError(t, err)

	running := fixtures.NewRunningExecution("still-running")
	err = repo.Insert(ctx, running)
	require.NoError(t, err)

	results, err := repo.GetFinished(ctx, testworkflow.NewExecutionsFilter())
	require.NoError(t, err)

	passedFound := false
	failedFound := false
	for _, r := range results {
		if r.Id == passed.Id {
			passedFound = true
		}
		if r.Id == failed.Id {
			failedFound = true
		}
	}
	assert.True(t, passedFound, "passed execution should be in GetFinished results")
	assert.True(t, failedFound, "failed execution should be in GetFinished results")
}

func testGetUnassigned(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	queued := fixtures.NewQueuedExecution("unassigned-test")
	err := repo.Insert(ctx, queued)
	require.NoError(t, err)

	running := fixtures.NewExecution("assigned-test",
		fixtures.WithRunnerID("runner-1"),
		fixtures.WithStatus(testkube.RUNNING_TestWorkflowStatus),
	)
	err = repo.Insert(ctx, running)
	require.NoError(t, err)

	results, err := repo.GetUnassigned(ctx)
	require.NoError(t, err)

	found := false
	for _, r := range results {
		if r.Id == queued.Id {
			found = true
			break
		}
	}
	assert.True(t, found, "queued execution should be in GetUnassigned results")
}
