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

func testGetLatestByTestWorkflow(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wfName := "latest-test"

	exec1 := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.PASSED_TestWorkflowStatus),
		fixtures.WithNumber(1),
		fixtures.WithScheduledAt(time.Now().Add(-2*time.Second)),
	)
	err := repo.Insert(ctx, exec1)
	require.NoError(t, err)

	exec2 := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.FAILED_TestWorkflowStatus),
		fixtures.WithNumber(2),
		fixtures.WithScheduledAt(time.Now()),
	)
	err = repo.Insert(ctx, exec2)
	require.NoError(t, err)

	latest, err := repo.GetLatestByTestWorkflow(ctx, wfName, testworkflow.LatestSortByScheduledAt)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, exec2.Id, latest.Id)
}

func testGetLatestByTestWorkflows(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()
	wf1 := "latest-multi-1"
	wf2 := "latest-multi-2"

	exec1 := fixtures.NewExecution(wf1,
		fixtures.WithStatus(testkube.PASSED_TestWorkflowStatus),
		fixtures.WithNumber(1),
	)
	err := repo.Insert(ctx, exec1)
	require.NoError(t, err)

	exec2 := fixtures.NewExecution(wf2,
		fixtures.WithStatus(testkube.FAILED_TestWorkflowStatus),
		fixtures.WithNumber(1),
	)
	err = repo.Insert(ctx, exec2)
	require.NoError(t, err)

	summaries, err := repo.GetLatestByTestWorkflows(ctx, []string{wf1, wf2})
	require.NoError(t, err)
	assert.Len(t, summaries, 2)
}
