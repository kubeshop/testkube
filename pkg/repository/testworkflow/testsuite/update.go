package testsuite

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/test/fixtures"
)

func testUpdate(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("update-test",
		fixtures.WithTags(map[string]string{"env": "dev"}),
	)

	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	execution.Tags = map[string]string{"env": "prod"}
	execution.Namespace = "updated-ns"

	err = repo.Update(ctx, execution)
	require.NoError(t, err)

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)

	assert.Equal(t, "prod", got.Tags["env"])
	assert.Equal(t, "updated-ns", got.Namespace)
}

func testUpdateResult(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewQueuedExecution("update-result-test")
	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	newResult := &testkube.TestWorkflowResult{
		Status:          fixtures.Ptr(testkube.PASSED_TestWorkflowStatus),
		PredictedStatus: fixtures.Ptr(testkube.PASSED_TestWorkflowStatus),
		DurationMs:      5000,
		TotalDurationMs: 5000,
		PausedMs:        0,
	}

	err = repo.UpdateResult(ctx, execution.Id, newResult)
	require.NoError(t, err)

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)

	require.NotNil(t, got.Result)
	assert.Equal(t, testkube.PASSED_TestWorkflowStatus, *got.Result.Status)
	assert.Equal(t, int32(5000), got.Result.DurationMs)
}

func testUpdateResultStrict(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	runnerID := "strict-runner"
	execution := fixtures.NewExecution("strict-result-test",
		fixtures.WithRunnerID(runnerID),
		fixtures.WithStatus(testkube.RUNNING_TestWorkflowStatus),
	)

	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	newResult := &testkube.TestWorkflowResult{
		Status:          fixtures.Ptr(testkube.PASSED_TestWorkflowStatus),
		PredictedStatus: fixtures.Ptr(testkube.PASSED_TestWorkflowStatus),
		DurationMs:      3000,
		TotalDurationMs: 3000,
		PausedMs:        0,
	}

	updated, err := repo.UpdateResultStrict(ctx, execution.Id, runnerID, newResult)
	require.NoError(t, err)
	assert.True(t, updated)
}

func testFinishResultStrict(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	runnerID := "finish-runner"
	execution := fixtures.NewExecution("finish-strict-test",
		fixtures.WithRunnerID(runnerID),
		fixtures.WithStatus(testkube.RUNNING_TestWorkflowStatus),
	)

	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	finishResult := &testkube.TestWorkflowResult{
		Status:          fixtures.Ptr(testkube.PASSED_TestWorkflowStatus),
		PredictedStatus: fixtures.Ptr(testkube.PASSED_TestWorkflowStatus),
		DurationMs:      8000,
		TotalDurationMs: 8000,
		PausedMs:        0,
		FinishedAt:      time.Now(),
	}

	updated, err := repo.FinishResultStrict(ctx, execution.Id, runnerID, finishResult)
	require.NoError(t, err)
	assert.True(t, updated)

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)

	require.NotNil(t, got.Result)
	assert.Equal(t, testkube.PASSED_TestWorkflowStatus, *got.Result.Status)
}
