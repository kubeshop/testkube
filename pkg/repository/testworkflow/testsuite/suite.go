package testsuite

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

func RunRepositoryTests(t *testing.T, repo testworkflow.Repository) {
	t.Run("InsertAndGet", func(t *testing.T) { testInsertAndGet(t, repo) })
	t.Run("GetByNameAndTestWorkflow", func(t *testing.T) { testGetByNameAndTestWorkflow(t, repo) })
	t.Run("GetWithRunner", func(t *testing.T) { testGetWithRunner(t, repo) })
	t.Run("Update", func(t *testing.T) { testUpdate(t, repo) })
	t.Run("UpdateResult", func(t *testing.T) { testUpdateResult(t, repo) })
	t.Run("UpdateResultStrict", func(t *testing.T) { testUpdateResultStrict(t, repo) })
	t.Run("FinishResultStrict", func(t *testing.T) { testFinishResultStrict(t, repo) })
	t.Run("UpdateReport", func(t *testing.T) { testUpdateReport(t, repo) })
	t.Run("UpdateOutput", func(t *testing.T) { testUpdateOutput(t, repo) })
	t.Run("UpdateResourceAggregations", func(t *testing.T) { testUpdateResourceAggregations(t, repo) })
	t.Run("UpdateTags", func(t *testing.T) { testUpdateTags(t, repo) })
	t.Run("GetExecutionTags", func(t *testing.T) { testGetExecutionTags(t, repo) })
	t.Run("GetExecutions", func(t *testing.T) { testGetExecutions(t, repo) })
	t.Run("GetExecutionsByName", func(t *testing.T) { testGetExecutionsByName(t, repo) })
	t.Run("GetExecutionsByStatus", func(t *testing.T) { testGetExecutionsByStatus(t, repo) })
	t.Run("GetExecutionsByTags", func(t *testing.T) { testGetExecutionsByTags(t, repo) })
	t.Run("GetExecutionsPagination", func(t *testing.T) { testGetExecutionsPagination(t, repo) })
	t.Run("GetExecutionsSummary", func(t *testing.T) { testGetExecutionsSummary(t, repo) })
	t.Run("Count", func(t *testing.T) { testCount(t, repo) })
	t.Run("GetExecutionsTotals", func(t *testing.T) { testGetExecutionsTotals(t, repo) })
	t.Run("GetPreviousFinishedState", func(t *testing.T) { testGetPreviousFinishedState(t, repo) })
	t.Run("GetRunning", func(t *testing.T) { testGetRunning(t, repo) })
	t.Run("GetFinished", func(t *testing.T) { testGetFinished(t, repo) })
	t.Run("GetUnassigned", func(t *testing.T) { testGetUnassigned(t, repo) })
	t.Run("GetLatestByTestWorkflow", func(t *testing.T) { testGetLatestByTestWorkflow(t, repo) })
	t.Run("GetLatestByTestWorkflows", func(t *testing.T) { testGetLatestByTestWorkflows(t, repo) })
	t.Run("Init", func(t *testing.T) { testInit(t, repo) })
	t.Run("Assign", func(t *testing.T) { testAssign(t, repo) })
	t.Run("AbortIfQueued", func(t *testing.T) { testAbortIfQueued(t, repo) })
	t.Run("DeleteByTestWorkflow", func(t *testing.T) { testDeleteByTestWorkflow(t, repo) })
	t.Run("DeleteByTestWorkflows", func(t *testing.T) { testDeleteByTestWorkflows(t, repo) })
	t.Run("DeleteAll", func(t *testing.T) { testDeleteAll(t, repo) })
	t.Run("GetTestWorkflowMetrics", func(t *testing.T) { testGetTestWorkflowMetrics(t, repo) })
	t.Run("GetNextExecutionNumber", func(t *testing.T) { testGetNextExecutionNumber(t, repo) })
}
