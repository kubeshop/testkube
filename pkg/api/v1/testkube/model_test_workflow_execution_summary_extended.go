package testkube

import (
	"github.com/kubeshop/testkube/pkg/utils"
)

type TestWorkflowExecutionSummaries []TestWorkflowExecutionSummary

func (executions TestWorkflowExecutionSummaries) Table() (header []string, output [][]string) {
	header = []string{"Id", "Name", "Test Workflow Name", "Status", "Labels", "Tags"}

	for _, e := range executions {
		status := "unknown"
		if e.Result != nil && e.Result.Status != nil {
			status = string(*e.Result.Status)
		}

		output = append(output, []string{
			e.Id,
			e.Name,
			e.Workflow.Name,
			status,
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
