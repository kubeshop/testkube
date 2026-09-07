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

func testInsertAndGet(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("insert-get-test",
		fixtures.WithStatus(testkube.QUEUED_TestWorkflowStatus),
		fixtures.WithTags(map[string]string{"env": "test"}),
	)

	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)

	assert.Equal(t, execution.Id, got.Id)
	assert.Equal(t, execution.Name, got.Name)
	assert.Equal(t, execution.Workflow.Name, got.Workflow.Name)
	if execution.Result != nil && execution.Result.Status != nil {
		assert.Equal(t, *execution.Result.Status, *got.Result.Status)
	}
	assert.Equal(t, execution.Tags["env"], got.Tags["env"])
}

func testGetByNameAndTestWorkflow(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("byname-test",
		fixtures.WithNumber(1),
	)
	execution.TestWorkflowExecutionName = "byname-test"

	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	got, err := repo.GetByNameAndTestWorkflow(ctx, execution.TestWorkflowExecutionName, execution.Workflow.Name)
	require.NoError(t, err)

	assert.Equal(t, execution.Id, got.Id)
	assert.Equal(t, execution.TestWorkflowExecutionName, got.TestWorkflowExecutionName)
	assert.Equal(t, execution.Workflow.Name, got.Workflow.Name)
}

func testGetWithRunner(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	runnerID := "runner-123"
	execution := fixtures.NewExecution("runner-test",
		fixtures.WithRunnerID(runnerID),
		fixtures.WithStatus(testkube.RUNNING_TestWorkflowStatus),
	)

	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	got, err := repo.GetWithRunner(ctx, execution.Id, runnerID)
	require.NoError(t, err)

	assert.Equal(t, execution.Id, got.Id)
}

// testInsertAndGetRerun covers the field that carries a rerun selection from
// the scheduler to the runner.
//
// It has to survive a round trip through storage, because that round trip *is*
// the delivery mechanism: the scheduler writes it onto the execution and the
// runner reads it back when it starts the pod. If it were dropped here, a
// rerun would silently run the whole suite.
func testInsertAndGetRerun(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("insert-get-rerun-test",
		fixtures.WithStatus(testkube.QUEUED_TestWorkflowStatus),
	)
	execution.Rerun = &testkube.TestWorkflowRerun{
		ExecutionId: "exec-original",
		OnlyFailed:  true,
		TestCases:   []string{"tests.a::test_one", "tests.b::test_two"},
	}

	require.NoError(t, repo.Insert(ctx, execution))

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)

	require.NotNil(t, got.Rerun)
	assert.Equal(t, "exec-original", got.Rerun.ExecutionId)
	assert.True(t, got.Rerun.OnlyFailed)
	assert.Equal(t, []string{"tests.a::test_one", "tests.b::test_two"}, got.Rerun.TestCases)
}

// testInsertAndGetWithoutRerun keeps the ordinary case honest: an execution
// that is not a rerun must come back without one, rather than with an empty
// struct a reader could mistake for a selection.
func testInsertAndGetWithoutRerun(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("insert-get-no-rerun-test",
		fixtures.WithStatus(testkube.QUEUED_TestWorkflowStatus),
	)
	require.NoError(t, repo.Insert(ctx, execution))

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)
	assert.Nil(t, got.Rerun)
}
