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

// A rerun's lineage has to survive the round trip through storage, since it is
// what the pod reads back to resolve the "rerun" reference.
func testInsertAndGetLineage(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("insert-get-lineage-test",
		fixtures.WithStatus(testkube.QUEUED_TestWorkflowStatus),
	)
	execution.Lineage = &testkube.TestWorkflowExecutionLineage{
		BaseId:  "exec-base",
		RootId:  "exec-root",
		Attempt: 3,
	}

	require.NoError(t, repo.Insert(ctx, execution))

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)

	require.NotNil(t, got.Lineage)
	assert.Equal(t, "exec-base", got.Lineage.BaseId)
	assert.Equal(t, "exec-root", got.Lineage.RootId)
	assert.Equal(t, int32(3), got.Lineage.Attempt)
}

// An original run records a root and an attempt but no base. That row is still
// lineage and must not read back as nil, or every original run would look like
// one written before the feature existed.
func testInsertAndGetLineageForAnOriginalRun(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("insert-get-lineage-original-test",
		fixtures.WithStatus(testkube.QUEUED_TestWorkflowStatus),
	)
	execution.Lineage = &testkube.TestWorkflowExecutionLineage{
		RootId:  execution.Id,
		Attempt: 1,
	}

	require.NoError(t, repo.Insert(ctx, execution))

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)

	require.NotNil(t, got.Lineage)
	assert.Empty(t, got.Lineage.BaseId)
	assert.Equal(t, execution.Id, got.Lineage.RootId)
	assert.Equal(t, int32(1), got.Lineage.Attempt)
}

// An execution written before lineage existed carries none, and has to come back
// nil so that EffectiveLineage supplies the default rather than the row claiming
// attempt zero.
func testInsertAndGetWithoutLineage(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	execution := fixtures.NewExecution("insert-get-without-lineage-test",
		fixtures.WithStatus(testkube.QUEUED_TestWorkflowStatus),
	)

	require.NoError(t, repo.Insert(ctx, execution))

	got, err := repo.Get(ctx, execution.Id)
	require.NoError(t, err)

	assert.Nil(t, got.Lineage)
	assert.Equal(t, execution.Id, got.EffectiveLineage().RootId)
	assert.Equal(t, int32(1), got.EffectiveLineage().Attempt)
}

// The summary is what list views render, and the two backends project it
// differently - Mongo names each field it keeps, so a new one is silently
// dropped there while working on Postgres. This is the only test that catches
// that divergence.
func testExecutionsSummaryCarriesLineage(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	wfName := "summary-lineage-test"
	execution := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.PASSED_TestWorkflowStatus),
		fixtures.WithNumber(1),
	)
	execution.Lineage = &testkube.TestWorkflowExecutionLineage{
		BaseId:  "exec-base",
		RootId:  "exec-root",
		Attempt: 2,
	}
	require.NoError(t, repo.Insert(ctx, execution))

	summaries, err := repo.GetExecutionsSummary(ctx, testworkflow.NewExecutionsFilter().WithName(wfName))
	require.NoError(t, err)
	require.Len(t, summaries, 1)

	require.NotNil(t, summaries[0].Lineage)
	assert.Equal(t, "exec-base", summaries[0].Lineage.BaseId)
	assert.Equal(t, "exec-root", summaries[0].Lineage.RootId)
	assert.Equal(t, int32(2), summaries[0].Lineage.Attempt)
}
