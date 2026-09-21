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

	// Stored sparsely - the row carries three NULLs - but read back as the
	// original run it is, because every consumer is promised a lineage on every
	// execution and the API would otherwise omit it for legacy rows while the
	// pod synthesized one.
	require.NotNil(t, got.Lineage)
	assert.Empty(t, got.Lineage.BaseId)
	assert.Equal(t, execution.Id, got.Lineage.RootId)
	assert.Equal(t, int32(1), got.Lineage.Attempt)
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

// Every reader has to apply the default, not just the ones that were thought
// about. Mongo returns exactly what was written and each reader post-processes
// the documents itself, while the Postgres side supplies lineage inside its row
// converter and so cannot forget; that asymmetry is how several Mongo readers
// came to return nil lineage for a legacy row while Postgres returned the
// default for the same execution, making the execution API answer differently
// depending on the backend.
//
// So this walks the readers rather than trusting one of them: a row written
// before lineage existed goes in, and every way of getting it back out has to
// report the original run it is.
func testEveryReaderAppliesTheLineageDefault(t *testing.T, repo testworkflow.Repository) {
	ctx := context.Background()

	const wfName = "reader-lineage-default-test"

	// One finished and one running execution of the same workflow, because the
	// readers are split by status: GetFinished and GetRunning each see only one
	// of the two, and both have to normalize.
	finished := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.PASSED_TestWorkflowStatus),
		fixtures.WithNumber(1),
		fixtures.WithRunnerID("runner-1"),
	)
	running := fixtures.NewExecution(wfName,
		fixtures.WithStatus(testkube.RUNNING_TestWorkflowStatus),
		fixtures.WithNumber(2),
		fixtures.WithRunnerID("runner-1"),
	)
	require.NoError(t, repo.Insert(ctx, finished))
	require.NoError(t, repo.Insert(ctx, running))

	// Nothing above sets Lineage, so both rows are stored the way an execution
	// written before the columns existed is stored.
	assertDefault := func(t *testing.T, reader string, id string, lineage *testkube.TestWorkflowExecutionLineage) {
		t.Helper()
		require.NotNil(t, lineage, "%s returned no lineage; a legacy row must read back as an original run", reader)
		assert.Empty(t, lineage.BaseId, "%s: an original run has no base", reader)
		assert.Equal(t, id, lineage.RootId, "%s: an original run is its own chain root", reader)
		assert.Equal(t, int32(1), lineage.Attempt, "%s: an original run is attempt 1", reader)
	}

	find := func(t *testing.T, reader, id string, executions []testkube.TestWorkflowExecution) {
		t.Helper()
		for i := range executions {
			if executions[i].Id == id {
				assertDefault(t, reader, id, executions[i].Lineage)
				return
			}
		}
		t.Fatalf("%s did not return execution %s", reader, id)
	}

	got, err := repo.Get(ctx, finished.Id)
	require.NoError(t, err)
	assertDefault(t, "Get", finished.Id, got.Lineage)

	got, err = repo.GetWithRunner(ctx, finished.Id, "runner-1")
	require.NoError(t, err)
	assertDefault(t, "GetWithRunner", finished.Id, got.Lineage)

	got, err = repo.GetByNameAndTestWorkflow(ctx, finished.Name, wfName)
	require.NoError(t, err)
	assertDefault(t, "GetByNameAndTestWorkflow", finished.Id, got.Lineage)

	latest, err := repo.GetLatestByTestWorkflow(ctx, wfName, testworkflow.LatestSortByNumber)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assertDefault(t, "GetLatestByTestWorkflow", latest.Id, latest.Lineage)

	summaries, err := repo.GetLatestByTestWorkflows(ctx, []string{wfName})
	require.NoError(t, err)
	require.NotEmpty(t, summaries)
	assertDefault(t, "GetLatestByTestWorkflows", summaries[0].Id, summaries[0].Lineage)

	executions, err := repo.GetExecutions(ctx, testworkflow.NewExecutionsFilter().WithName(wfName))
	require.NoError(t, err)
	find(t, "GetExecutions", finished.Id, executions)

	executions, err = repo.GetFinished(ctx, testworkflow.NewExecutionsFilter().WithName(wfName))
	require.NoError(t, err)
	find(t, "GetFinished", finished.Id, executions)

	executions, err = repo.GetRunning(ctx)
	require.NoError(t, err)
	find(t, "GetRunning", running.Id, executions)

	summaries, err = repo.GetExecutionsSummary(ctx, testworkflow.NewExecutionsFilter().WithName(wfName))
	require.NoError(t, err)
	require.NotEmpty(t, summaries)
	for i := range summaries {
		if summaries[i].Id == finished.Id {
			assertDefault(t, "GetExecutionsSummary", finished.Id, summaries[i].Lineage)
		}
	}
}
