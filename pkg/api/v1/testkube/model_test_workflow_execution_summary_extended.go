package testkube

import (
	"github.com/kubeshop/testkube/pkg/utils"
)

type TestWorkflowExecutionSummaries []TestWorkflowExecutionSummary

func (executions TestWorkflowExecutionSummaries) Table() (header []string, output [][]string) {
	header = []string{"Id", "Name", "Test Workflow Name", "Status", "Reason", "Labels", "Tags"}

	for _, e := range executions {
		status := "unknown"
		reason := ""
		if e.Result != nil && e.Result.Status != nil {
			status = string(*e.Result.Status)
			reason = e.Result.StatusDetails.Label()
		}

		output = append(output, []string{
			e.Id,
			e.Name,
			e.Workflow.Name,
			status,
			reason,
			MapToString(e.Workflow.Labels),
			MapToString(e.Tags),
		})
	}

	return
}

func (e *TestWorkflowExecutionSummary) ConvertDots(fn func(string) string) *TestWorkflowExecutionSummary {
	e.Workflow.ConvertDots(fn)
	if e.Tags != nil {
		e.Tags = convertDotsInMap(e.Tags, fn)
	}
	return e
}

func (e *TestWorkflowExecutionSummary) EscapeDots() *TestWorkflowExecutionSummary {
	return e.ConvertDots(utils.EscapeDots)
}

func (e *TestWorkflowExecutionSummary) UnscapeDots() *TestWorkflowExecutionSummary {
	return e.ConvertDots(utils.UnescapeDots)
}

// ApplyEffectiveLineage fills in the lineage a reader should see, matching
// TestWorkflowExecution.ApplyEffectiveLineage: a summary for an execution
// recorded before the columns existed still reports it as its own root at
// attempt 1, so a list view can group a chain without special-casing them.
func (e *TestWorkflowExecutionSummary) ApplyEffectiveLineage() {
	if e == nil {
		return
	}
	if e.Lineage != nil {
		lineage := *e.Lineage
		if lineage.RootId == "" {
			lineage.RootId = e.Id
		}
		if lineage.Attempt == 0 {
			lineage.Attempt = 1
		}
		e.Lineage = &lineage
		return
	}
	e.Lineage = &TestWorkflowExecutionLineage{RootId: e.Id, Attempt: 1}
}
