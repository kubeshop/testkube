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

func testInit(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewQueuedExecution("init-test")
	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	data := testworkflow.InitData{
		RunnerID:  "init-runner-1",
		Namespace: "test-ns",
		Signature: []testkube.TestWorkflowSignature{
			{Ref: "step-1"},
		},
	}
	err = repo.Init(ctx, execution.Id, data)
	require.NoError(t, err)

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)

	assert.Equal(t, "init-runner-1", got.RunnerId)
	assert.Equal(t, "test-ns", got.Namespace)
}

func testAssign(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewQueuedExecution("assign-test")
	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	assigned, err := repo.Assign(ctx, execution.Id, "", "runner-assign-1", nil)
	require.NoError(t, err)
	assert.True(t, assigned)

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)
	assert.Equal(t, "runner-assign-1", got.RunnerId)
}

func testAbortIfQueued(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("abort-queued-test",
		fixtures.WithStatus(testkube.QUEUED_TestWorkflowStatus),
		fixtures.WithInitialization(&testkube.TestWorkflowStepResult{
			Status: fixtures.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
		}),
	)
	err := repo.Insert(ctx, execution)
	require.NoError(t, err)

	aborted, err := repo.AbortIfQueued(ctx, execution.Id)
	require.NoError(t, err)
	assert.True(t, aborted)

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)
	require.NotNil(t, got.Result)
	assert.Equal(t, testkube.ABORTED_TestWorkflowStatus, *got.Result.Status)
}
