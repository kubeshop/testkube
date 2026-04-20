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
