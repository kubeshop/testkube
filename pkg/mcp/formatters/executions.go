package formatters

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// formattedExecutionsResult is a compact representation of execution list results.
type formattedExecutionsResult struct {
	Totals   *testkube.ExecutionsTotals `json:"totals,omitempty"`
	Filtered *testkube.ExecutionsTotals `json:"filtered,omitempty"`
	Results  []formattedExecution       `json:"results"`
}

// formattedExecution is a compact representation of an execution for MCP responses.
type formattedExecution struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Number       int32     `json:"number,omitempty"`
	ScheduledAt  time.Time `json:"scheduledAt,omitempty"`
	Status       string    `json:"status,omitempty"`
	Duration     string    `json:"duration,omitempty"`
	WorkflowName string    `json:"workflowName,omitempty"`
	ActorType    string    `json:"actorType,omitempty"`
	ActorName    string    `json:"actorName,omitempty"`
}

// FormatListExecutions parses a raw API response (JSON or YAML) containing
// testkube.TestWorkflowExecutionsResult and returns a compact JSON
// representation with only essential fields.
// It strips resourceAggregations, configParams, full workflow object,
// and verbose runningContext (keeping only actor.type and actor.name).
func FormatListExecutions(raw string) (string, error) {
	result, isEmpty, err := ParseJSON[testkube.TestWorkflowExecutionsResult](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "{}", nil
	}

	formatted := formattedExecutionsResult{
		Totals:   result.Totals,
		Filtered: result.Filtered,
		Results:  make([]formattedExecution, 0, len(result.Results)),
	}

	for _, exec := range result.Results {
		f := formattedExecution{
			ID:          exec.Id,
			Name:        exec.Name,
			Number:      exec.Number,
			ScheduledAt: exec.ScheduledAt,
		}

		// Extract workflow name from workflow summary
		if exec.Workflow != nil {
			f.WorkflowName = exec.Workflow.Name
		}

		// Extract status and duration from result
		if exec.Result != nil {
			if exec.Result.Status != nil {
				f.Status = string(*exec.Result.Status)
			}
			f.Duration = exec.Result.Duration
		}

		// Extract only actor type and name from running context
		if exec.RunningContext != nil && exec.RunningContext.Actor != nil {
			f.ActorName = exec.RunningContext.Actor.Name
			if exec.RunningContext.Actor.Type_ != nil {
				f.ActorType = string(*exec.RunningContext.Actor.Type_)
			}
		}

		formatted.Results = append(formatted.Results, f)
	}

	return FormatJSON(formatted)
}
