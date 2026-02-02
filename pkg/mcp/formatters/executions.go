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

// formattedExecutionInfo is a compact representation of execution details.
// It strips the verbose workflow/resolvedWorkflow specs, detailed output array,
// and step-level timing, keeping only essential status and identification info.
type formattedExecutionInfo struct {
	ID           string                    `json:"id"`
	Name         string                    `json:"name"`
	Number       int32                     `json:"number,omitempty"`
	Namespace    string                    `json:"namespace,omitempty"`
	ScheduledAt  time.Time                 `json:"scheduledAt,omitempty"`
	WorkflowName string                    `json:"workflowName,omitempty"`
	Result       *formattedExecutionResult `json:"result,omitempty"`
	Signature    []formattedSignature      `json:"signature,omitempty"`
	ConfigParams map[string]string         `json:"configParams,omitempty"`
	ActorType    string                    `json:"actorType,omitempty"`
	ActorName    string                    `json:"actorName,omitempty"`
	Tags         map[string]string         `json:"tags,omitempty"`
}

// formattedExecutionResult is a compact representation of execution result.
type formattedExecutionResult struct {
	Status          string    `json:"status,omitempty"`
	PredictedStatus string    `json:"predictedStatus,omitempty"`
	Duration        string    `json:"duration,omitempty"`
	TotalDuration   string    `json:"totalDuration,omitempty"`
	QueuedAt        time.Time `json:"queuedAt,omitempty"`
	StartedAt       time.Time `json:"startedAt,omitempty"`
	FinishedAt      time.Time `json:"finishedAt,omitempty"`
}

// formattedSignature is a compact representation of workflow step signature.
type formattedSignature struct {
	Ref      string `json:"ref"`
	Name     string `json:"name,omitempty"`
	Category string `json:"category,omitempty"`
}

// FormatExecutionInfo parses a raw API response (JSON or YAML) containing
// testkube.TestWorkflowExecution and returns a compact JSON representation.
// It strips: workflow spec, resolvedWorkflow spec, output array details,
// step-level timing in result.steps, reports, resourceAggregations.
func FormatExecutionInfo(raw string) (string, error) {
	exec, isEmpty, err := ParseJSON[testkube.TestWorkflowExecution](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "{}", nil
	}

	formatted := formattedExecutionInfo{
		ID:          exec.Id,
		Name:        exec.Name,
		Number:      exec.Number,
		Namespace:   exec.Namespace,
		ScheduledAt: exec.ScheduledAt,
		Tags:        exec.Tags,
	}

	// Extract workflow name
	if exec.Workflow != nil {
		formatted.WorkflowName = exec.Workflow.Name
	}

	// Extract result without step-level details
	if exec.Result != nil {
		formatted.Result = &formattedExecutionResult{
			Duration:      exec.Result.Duration,
			TotalDuration: exec.Result.TotalDuration,
			QueuedAt:      exec.Result.QueuedAt,
			StartedAt:     exec.Result.StartedAt,
			FinishedAt:    exec.Result.FinishedAt,
		}
		if exec.Result.Status != nil {
			formatted.Result.Status = string(*exec.Result.Status)
		}
		if exec.Result.PredictedStatus != nil {
			formatted.Result.PredictedStatus = string(*exec.Result.PredictedStatus)
		}
	}

	// Extract signature (step structure without details)
	if len(exec.Signature) > 0 {
		formatted.Signature = make([]formattedSignature, 0, len(exec.Signature))
		for _, sig := range exec.Signature {
			formatted.Signature = append(formatted.Signature, formattedSignature{
				Ref:      sig.Ref,
				Name:     sig.Name,
				Category: sig.Category,
			})
		}
	}

	// Extract config param values only (not full structure)
	if len(exec.ConfigParams) > 0 {
		formatted.ConfigParams = make(map[string]string)
		for key, param := range exec.ConfigParams {
			if param.Value != "" {
				formatted.ConfigParams[key] = param.Value
			} else if param.DefaultValue != "" {
				formatted.ConfigParams[key] = param.DefaultValue
			}
		}
	}

	// Extract actor info
	if exec.RunningContext != nil && exec.RunningContext.Actor != nil {
		formatted.ActorName = exec.RunningContext.Actor.Name
		if exec.RunningContext.Actor.Type_ != nil {
			formatted.ActorType = string(*exec.RunningContext.Actor.Type_)
		}
	}

	return FormatJSON(formatted)
}

// formattedAbortResult is a compact representation of abort execution result.
type formattedAbortResult struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
}

// FormatAbortExecution parses the abort workflow execution response.
// The abort response returns the execution state after aborting.
// We extract only the essential fields: id, name, and resulting status.
func FormatAbortExecution(raw string) (string, error) {
	exec, isEmpty, err := ParseJSON[testkube.TestWorkflowExecution](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "{}", nil
	}

	formatted := formattedAbortResult{
		ID:   exec.Id,
		Name: exec.Name,
	}

	if exec.Result != nil && exec.Result.Status != nil {
		formatted.Status = string(*exec.Result.Status)
	}

	return FormatJSON(formatted)
}

// formattedWaitResult contains the results of waiting for multiple executions.
type formattedWaitResult struct {
	Executions []formattedWaitExecution `json:"executions"`
}

// formattedWaitExecution is a compact representation of execution status after waiting.
type formattedWaitExecution struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// FormatWaitForExecutions parses the wait for executions response.
// The response is an array of execution results. We extract only
// the essential fields for each: id, name, status, and duration.
func FormatWaitForExecutions(raw string) (string, error) {
	executions, isEmpty, err := ParseJSON[[]testkube.TestWorkflowExecution](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "[]", nil
	}

	formatted := formattedWaitResult{
		Executions: make([]formattedWaitExecution, 0, len(executions)),
	}

	for _, exec := range executions {
		f := formattedWaitExecution{
			ID:   exec.Id,
			Name: exec.Name,
		}

		if exec.Result != nil {
			if exec.Result.Status != nil {
				f.Status = string(*exec.Result.Status)
			}
			f.Duration = exec.Result.Duration
		}

		formatted.Executions = append(formatted.Executions, f)
	}

	return FormatJSON(formatted)
}
