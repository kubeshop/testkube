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

func testUpdateTags(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("tags-test",
		fixtures.WithTags(map[string]string{"env": "dev", "team": "backend"}),
	)
	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	newTags := map[string]string{"env": "prod", "priority": "high"}
	err = repo.UpdateTags(ctx, execution.Id, newTags)
	require.NoError(t, err)

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)
	assert.Equal(t, "prod", got.Tags["env"])
	assert.Equal(t, "high", got.Tags["priority"])
	_, hasTeam := got.Tags["team"]
	assert.False(t, hasTeam, "old tag 'team' should be removed after UpdateTags")
}

func testGetExecutionTags(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	wfName := "tags-wf-" + time.Now().Format("20060102150405")
	for i := 0; i < 2; i++ {
		execution := fixtures.NewExecution(wfName,
			fixtures.WithTags(map[string]string{"env": "dev", "team": "backend"}),
			fixtures.WithNumber(int32(i+1)),
		)
		err := repo.Insert(ctx, execution)
		require.NoError(t, err)
	}

	tags, err := repo.GetExecutionTags(ctx, wfName)
	require.NoError(t, err)

	assert.Contains(t, tags, "env")
	assert.Contains(t, tags, "team")
	envVals := tags["env"]
	assert.Contains(t, envVals, "dev")
}

func testGetExecutions(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "filter-test"

	exec1 := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.PASSED_TestWorkflowStatus),
		fixtures.WithTags(map[string]string{"env": "prod"}),
		fixtures.WithNumber(1),
	)
	exec2 := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.FAILED_TestWorkflowStatus),
		fixtures.WithTags(map[string]string{"env": "dev"}),
		fixtures.WithNumber(2),
	)
	exec3 := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.RUNNING_TestWorkflowStatus),
		fixtures.WithNumber(3),
	)

	for _, e := range []testkube.TestWorkflowExecution{exec1, exec2, exec3} {
		err := repo.Insert(ctx, e)
		require.NoError(t, err)
	}

	filter := testworkflow.NewExecutionsFilter().WithName(wfName)
	results, err := repo.GetExecutions(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func testGetExecutionsByName(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "name-filter-test"

	exec1 := fixtures.NewExecution(wfName, fixtures.WithNumber(1))
	exec2 := fixtures.NewExecution(wfName, fixtures.WithNumber(2))
	execOther := fixtures.NewExecution("other-workflow", fixtures.WithNumber(1))

	for _, e := range []testkube.TestWorkflowExecution{exec1, exec2, execOther} {
		err := repo.Insert(ctx, e)
		require.NoError(t, err)
	}

	filter := testworkflow.NewExecutionsFilter().WithName(wfName)
	results, err := repo.GetExecutions(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func testGetExecutionsByStatus(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "status-filter-test"

	exec1 := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.PASSED_TestWorkflowStatus),
		fixtures.WithNumber(1),
	)
	exec2 := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.PASSED_TestWorkflowStatus),
		fixtures.WithNumber(2),
	)
	exec3 := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.FAILED_TestWorkflowStatus),
		fixtures.WithNumber(3),
	)

	for _, e := range []testkube.TestWorkflowExecution{exec1, exec2, exec3} {
		err := repo.Insert(ctx, e)
		require.NoError(t, err)
	}

	filter := testworkflow.NewExecutionsFilter().WithName(wfName).WithStatus("passed")
	results, err := repo.GetExecutions(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func testGetExecutionsByTags(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "tag-filter-test"

	exec1 := fixtures.NewExecution(wfName,
		fixtures.WithTags(map[string]string{"env": "prod"}),
		fixtures.WithNumber(1),
	)
	exec2 := fixtures.NewExecution(wfName,
		fixtures.WithTags(map[string]string{"env": "dev"}),
		fixtures.WithNumber(2),
	)

	for _, e := range []testkube.TestWorkflowExecution{exec1, exec2} {
		err := repo.Insert(ctx, e)
		require.NoError(t, err)
	}

	filter := testworkflow.NewExecutionsFilter().WithName(wfName).WithTagSelector("env=prod")
	results, err := repo.GetExecutions(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	if len(results) > 0 {
		assert.Equal(t, "prod", results[0].Tags["env"])
	}
}

func testGetExecutionsPagination(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "pagination-test"

	for i := 0; i < 5; i++ {
		exec := fixtures.NewExecution(wfName, fixtures.WithNumber(int32(i+1)))
		err := repo.Insert(ctx, exec)
		require.NoError(t, err)
	}

	filter := testworkflow.NewExecutionsFilter().WithName(wfName).WithPage(0).WithPageSize(2)
	results, err := repo.GetExecutions(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	filter = testworkflow.NewExecutionsFilter().WithName(wfName).WithPage(1).WithPageSize(2)
	results, err = repo.GetExecutions(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	filter = testworkflow.NewExecutionsFilter().WithName(wfName).WithPage(2).WithPageSize(2)
	results, err = repo.GetExecutions(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func testGetExecutionsSummary(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "summary-test"

	exec := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.PASSED_TestWorkflowStatus),
		fixtures.WithNumber(1),
	)
	err := repo.Insert(ctx, exec)
	require.NoError(t, err)

	filter := testworkflow.NewExecutionsFilter().WithName(wfName)
	summaries, err := repo.GetExecutionsSummary(ctx, filter)
	require.NoError(t, err)
	require.Len(t, summaries, 1)

	assert.Equal(t, exec.Id, summaries[0].Id)
	assert.Equal(t, exec.Name, summaries[0].Name)
	require.NotNil(t, summaries[0].Result)
	assert.Equal(t, testkube.PASSED_TestWorkflowStatus, *summaries[0].Result.Status)
}

func testCount(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "count-test"

	for i := 0; i < 3; i++ {
		exec := fixtures.NewExecution(wfName, fixtures.WithNumber(int32(i+1)))
		err := repo.Insert(ctx, exec)
		require.NoError(t, err)
	}

	filter := testworkflow.NewExecutionsFilter().WithName(wfName)
	count, err := repo.Count(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func testGetExecutionsTotals(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "totals-test"

	for _, status := range []testkube.TestWorkflowStatus{
		testkube.PASSED_TestWorkflowStatus,
		testkube.PASSED_TestWorkflowStatus,
		testkube.FAILED_TestWorkflowStatus,
		testkube.QUEUED_TestWorkflowStatus,
		testkube.RUNNING_TestWorkflowStatus,
	} {
		exec := fixtures.NewExecution(wfName, fixtures.WithStatus(status))
		err := repo.Insert(ctx, exec)
		require.NoError(t, err)
	}

	totals, err := repo.GetExecutionsTotals(ctx, testworkflow.NewExecutionsFilter().WithName(wfName))
	require.NoError(t, err)

	assert.Equal(t, int32(5), totals.Results)
	assert.Equal(t, int32(2), totals.Passed)
	assert.Equal(t, int32(1), totals.Failed)
}

func testGetPreviousFinishedState(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "prev-state-test"

	exec := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.FAILED_TestWorkflowStatus),
		fixtures.WithNumber(1),
	)
	err := repo.Insert(ctx, exec)
	require.NoError(t, err)

	status, err := repo.GetPreviousFinishedState(ctx, wfName, time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, testkube.FAILED_TestWorkflowStatus, status)
}
