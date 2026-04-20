package testsuite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/test/fixtures"
)

func testDeleteByTestWorkflow(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "delete-wf-test"

	for i := 0; i < 3; i++ {
		exec := fixtures.NewExecution(wfName, fixtures.WithNumber(int32(i+1)))
		err := repo.Insert(ctx, exec)
		require.NoError(t, err)
	}

	otherExec := fixtures.NewExecution("other-wf")
	err := repo.Insert(ctx, otherExec)
	require.NoError(t, err)

	err = repo.DeleteByTestWorkflow(ctx, wfName)
	require.NoError(t, err)

	filter := testworkflow.NewExecutionsFilter().WithName(wfName)
	results, err := repo.GetExecutions(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, results, 0)

	_, err = repo.Get(ctx, otherExec.Id)
	require.NoError(t, err)
}

func testDeleteByTestWorkflows(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wf1 := "delete-multi-wf-1"
	wf2 := "delete-multi-wf-2"

	for _, wf := range []string{wf1, wf2} {
		exec := fixtures.NewExecution(wf)
		err := repo.Insert(ctx, exec)
		require.NoError(t, err)
	}

	keepExec := fixtures.NewExecution("keep-wf")
	err := repo.Insert(ctx, keepExec)
	require.NoError(t, err)

	err = repo.DeleteByTestWorkflows(ctx, []string{wf1, wf2})
	require.NoError(t, err)

	_, err = repo.Get(ctx, keepExec.Id)
	require.NoError(t, err)
}

func testDeleteAll(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		exec := fixtures.NewExecution("delete-all-test")
		err := repo.Insert(ctx, exec)
		require.NoError(t, err)
	}

	err := repo.DeleteAll(ctx)
	require.NoError(t, err)

	results, err := repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter())
	require.NoError(t, err)
	assert.Len(t, results, 0)
}
