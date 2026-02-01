package formatters

import (
	"encoding/json"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// formattedWorkflow is a compact representation of a workflow for MCP responses.
type formattedWorkflow struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Created     time.Time         `json:"created,omitempty"`
	Updated     time.Time         `json:"updated,omitempty"`
	Health      *formattedHealth  `json:"health,omitempty"`
	Latest      *formattedLatest  `json:"latest,omitempty"`
}

// formattedHealth contains workflow health metrics.
type formattedHealth struct {
	PassRate      float64 `json:"passRate"`
	FlipRate      float64 `json:"flipRate"`
	OverallHealth float64 `json:"overallHealth"`
}

// formattedLatest contains the latest execution summary.
type formattedLatest struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Number      int32     `json:"number,omitempty"`
	Status      string    `json:"status,omitempty"`
	ScheduledAt time.Time `json:"scheduledAt,omitempty"`
	Duration    string    `json:"duration,omitempty"`
}

// FormatListWorkflows parses a raw API response (JSON or YAML) containing
// []testkube.TestWorkflowWithExecutionSummary and returns a compact JSON
// representation with only essential fields.
func FormatListWorkflows(raw string) (string, error) {
	workflows, isEmpty, err := ParseJSON[[]testkube.TestWorkflowWithExecutionSummary](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "[]", nil
	}

	formatted := make([]formattedWorkflow, 0, len(workflows))
	for _, wf := range workflows {
		f := formattedWorkflow{}

		if wf.Workflow != nil {
			f.Name = wf.Workflow.Name
			f.Namespace = wf.Workflow.Namespace
			f.Description = wf.Workflow.Description
			f.Labels = wf.Workflow.Labels
			f.Created = wf.Workflow.Created
			f.Updated = wf.Workflow.Updated

			if wf.Workflow.Status != nil && wf.Workflow.Status.Health != nil {
				f.Health = &formattedHealth{
					PassRate:      wf.Workflow.Status.Health.PassRate,
					FlipRate:      wf.Workflow.Status.Health.FlipRate,
					OverallHealth: wf.Workflow.Status.Health.OverallHealth,
				}
			}
		}

		if wf.LatestExecution != nil {
			f.Latest = &formattedLatest{
				ID:          wf.LatestExecution.Id,
				Name:        wf.LatestExecution.Name,
				Number:      wf.LatestExecution.Number,
				ScheduledAt: wf.LatestExecution.ScheduledAt,
			}

			if wf.LatestExecution.Result != nil {
				if wf.LatestExecution.Result.Status != nil {
					f.Latest.Status = string(*wf.LatestExecution.Result.Status)
				}
				f.Latest.Duration = wf.LatestExecution.Result.Duration
			}
		}

		formatted = append(formatted, f)
	}

	return FormatJSON(formatted)
}

// formattedWorkflowDetails is a compact representation of a single workflow for GetWorkflow.
// It includes essential metadata, health status, and the spec for understanding the workflow.
type formattedWorkflowDetails struct {
	Name        string                     `json:"name"`
	Namespace   string                     `json:"namespace,omitempty"`
	Description string                     `json:"description,omitempty"`
	Labels      map[string]string          `json:"labels,omitempty"`
	Created     time.Time                  `json:"created,omitempty"`
	Updated     time.Time                  `json:"updated,omitempty"`
	Health      *formattedHealth           `json:"health,omitempty"`
	Spec        *testkube.TestWorkflowSpec `json:"spec,omitempty"`
	ReadOnly    bool                       `json:"readOnly,omitempty"`
}

// FormatGetWorkflow parses a raw API response (JSON or YAML) containing
// either testkube.TestWorkflow or testkube.TestWorkflowWithExecutionSummary
// and returns a compact JSON representation.
// It preserves the spec (needed for understanding the workflow) but strips annotations.
func FormatGetWorkflow(raw string) (string, error) {
	// First try parsing as TestWorkflowWithExecutionSummary (the format returned by
	// /agent/test-workflow-with-executions/{workflowName})
	wfWithExec, isEmpty, err := ParseJSON[testkube.TestWorkflowWithExecutionSummary](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "{}", nil
	}

	// If the response has a workflow wrapper, use it
	var wf *testkube.TestWorkflow
	if wfWithExec.Workflow != nil {
		wf = wfWithExec.Workflow
	} else {
		// Fallback: try parsing directly as TestWorkflow (for backward compatibility)
		directWf, _, err := ParseJSON[testkube.TestWorkflow](raw)
		if err != nil {
			return "", err
		}
		wf = &directWf
	}

	formatted := formattedWorkflowDetails{
		Name:        wf.Name,
		Namespace:   wf.Namespace,
		Description: wf.Description,
		Labels:      wf.Labels,
		Created:     wf.Created,
		Updated:     wf.Updated,
		Spec:        wf.Spec,
		ReadOnly:    wf.ReadOnly,
	}

	if wf.Status != nil && wf.Status.Health != nil {
		formatted.Health = &formattedHealth{
			PassRate:      wf.Status.Health.PassRate,
			FlipRate:      wf.Status.Health.FlipRate,
			OverallHealth: wf.Status.Health.OverallHealth,
		}
	}

	return FormatJSON(formatted)
}

// FormatGetWorkflowDefinition is a pass-through formatter for workflow definitions.
// The full YAML/JSON definition is needed by AI to understand the complete workflow
// structure including all steps, configuration schema, and templates.
// No fields are stripped since AI needs the complete spec for analysis and modification.
func FormatGetWorkflowDefinition(raw string) (string, error) {
	// Validate that input is parseable, but return as-is
	_, isEmpty, err := ParseJSON[testkube.TestWorkflow](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "{}", nil
	}
	// Return unchanged - AI needs full definition for workflow analysis
	return raw, nil
}

// formattedRunWorkflowResult is a compact representation of workflow execution start result.
type formattedRunWorkflowResult struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Number       int32     `json:"number,omitempty"`
	Namespace    string    `json:"namespace,omitempty"`
	ScheduledAt  time.Time `json:"scheduledAt,omitempty"`
	WorkflowName string    `json:"workflowName,omitempty"`
	Status       string    `json:"status,omitempty"`
}

// FormatRunWorkflow parses a raw API response containing the execution created
// when running a workflow. Returns a compact JSON with essential fields.
// Strips: workflow spec, resolvedWorkflow, signature details, output, reports.
func FormatRunWorkflow(raw string) (string, error) {
	exec, isEmpty, err := ParseJSON[testkube.TestWorkflowExecution](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "{}", nil
	}

	formatted := formattedRunWorkflowResult{
		ID:          exec.Id,
		Name:        exec.Name,
		Number:      exec.Number,
		Namespace:   exec.Namespace,
		ScheduledAt: exec.ScheduledAt,
	}

	if exec.Workflow != nil {
		formatted.WorkflowName = exec.Workflow.Name
	}

	if exec.Result != nil && exec.Result.Status != nil {
		formatted.Status = string(*exec.Result.Status)
	}

	return FormatJSON(formatted)
}

// formattedCreateWorkflowResult is a compact representation of workflow creation result.
type formattedCreateWorkflowResult struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Created     time.Time         `json:"created,omitempty"`
}

// FormatCreateWorkflow parses a raw API response containing the created workflow.
// Returns a compact JSON confirming the workflow was created with key metadata.
// Strips: full spec (already known by caller), annotations, status.
func FormatCreateWorkflow(raw string) (string, error) {
	wf, isEmpty, err := ParseJSON[testkube.TestWorkflow](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "{}", nil
	}

	formatted := formattedCreateWorkflowResult{
		Name:        wf.Name,
		Namespace:   wf.Namespace,
		Description: wf.Description,
		Labels:      wf.Labels,
		Created:     wf.Created,
	}

	return FormatJSON(formatted)
}

// FormatUpdateWorkflow parses a raw API response containing the updated workflow.
// Uses the same compact format as FormatCreateWorkflow.
// Strips: full spec (already known by caller), annotations, status.
func FormatUpdateWorkflow(raw string) (string, error) {
	return FormatCreateWorkflow(raw)
}

// formattedWorkflowMetrics is a compact representation of workflow metrics.
type formattedWorkflowMetrics struct {
	PassFailRatio        float64 `json:"passFailRatio,omitempty"`
	TotalExecutions      int     `json:"totalExecutions,omitempty"`
	ExecutionDurationP50 string  `json:"executionDurationP50,omitempty"`
	ExecutionDurationP90 string  `json:"executionDurationP90,omitempty"`
	ExecutionDurationP95 string  `json:"executionDurationP95,omitempty"`
	ExecutionDurationP99 string  `json:"executionDurationP99,omitempty"`
}

// workflowMetricsResponse mirrors the API response for workflow metrics.
type workflowMetricsResponse struct {
	PassFailRatio        float64 `json:"passFailRatio"`
	TotalExecutions      int     `json:"totalExecutions"`
	ExecutionDurationP50 string  `json:"executionDurationP50"`
	ExecutionDurationP90 string  `json:"executionDurationP90"`
	ExecutionDurationP95 string  `json:"executionDurationP95"`
	ExecutionDurationP99 string  `json:"executionDurationP99"`
	// Executions array is stripped - it's verbose and execution details
	// can be obtained via list_executions if needed
}

// FormatGetWorkflowMetrics parses a raw API response containing workflow metrics.
// Returns a compact JSON with aggregate metrics only.
// Strips: executions array (verbose, available via list_executions tool).
func FormatGetWorkflowMetrics(raw string) (string, error) {
	metrics, isEmpty, err := ParseJSON[workflowMetricsResponse](raw)
	if err != nil {
		return "", err
	}
	if isEmpty {
		return "{}", nil
	}

	formatted := formattedWorkflowMetrics{
		PassFailRatio:        metrics.PassFailRatio,
		TotalExecutions:      metrics.TotalExecutions,
		ExecutionDurationP50: metrics.ExecutionDurationP50,
		ExecutionDurationP90: metrics.ExecutionDurationP90,
		ExecutionDurationP95: metrics.ExecutionDurationP95,
		ExecutionDurationP99: metrics.ExecutionDurationP99,
	}

	return FormatJSON(formatted)
}

// formattedExecutionMetrics is a compact representation of execution resource metrics.
type formattedExecutionMetrics struct {
	ExecutionID string                  `json:"executionId,omitempty"`
	Global      *formattedResourceStats `json:"global,omitempty"`
}

// formattedResourceStats contains summarized resource usage.
type formattedResourceStats struct {
	CPUMillicores   *formattedMetricStat `json:"cpuMillicores,omitempty"`
	MemoryUsedBytes *formattedMetricStat `json:"memoryUsedBytes,omitempty"`
	NetworkRecvKBps *formattedMetricStat `json:"networkRecvKBps,omitempty"`
	NetworkSentKBps *formattedMetricStat `json:"networkSentKBps,omitempty"`
}

// formattedMetricStat contains min/max/avg for a metric.
type formattedMetricStat struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Avg float64 `json:"avg"`
}

// executionMetricsResponse mirrors the API response for execution metrics.
type executionMetricsResponse struct {
	ResourceAggregations *resourceAggregations `json:"resourceAggregations,omitempty"`
}

type resourceAggregations struct {
	Global *resourceGlobal `json:"global,omitempty"`
}

type resourceGlobal struct {
	CPU     *resourceCPU     `json:"cpu,omitempty"`
	Memory  *resourceMemory  `json:"memory,omitempty"`
	Network *resourceNetwork `json:"network,omitempty"`
}

type resourceCPU struct {
	Millicores *resourceMetric `json:"millicores,omitempty"`
}

type resourceMemory struct {
	Used *resourceMetric `json:"used,omitempty"`
}

type resourceNetwork struct {
	BytesRecvPerS *resourceMetric `json:"bytes_recv_per_s,omitempty"`
	BytesSentPerS *resourceMetric `json:"bytes_sent_per_s,omitempty"`
}

type resourceMetric struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Avg float64 `json:"avg"`
}

// FormatGetWorkflowExecutionMetrics parses a raw API response containing execution metrics.
// Returns a compact JSON with global resource usage summary.
// Strips: per-step metrics (verbose), detailed disk I/O stats, totals and stdDev.
func FormatGetWorkflowExecutionMetrics(raw string) (string, error) {
	// First try to parse as the direct resourceAggregations structure
	// (which is what the execution info API returns)
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &rawData); err != nil {
		// If it fails, check if input is empty
		if IsEmptyInput(raw) {
			return "{}", nil
		}
		return "", err
	}

	if IsEmptyInput(raw) {
		return "{}", nil
	}

	// Check if we have resourceAggregations directly or nested
	var agg *resourceAggregations
	if raData, ok := rawData["resourceAggregations"]; ok {
		var ra resourceAggregations
		if err := json.Unmarshal(raData, &ra); err != nil {
			return "", err
		}
		agg = &ra
	} else if globalData, ok := rawData["global"]; ok {
		// The response IS the resourceAggregations object
		var global resourceGlobal
		if err := json.Unmarshal(globalData, &global); err != nil {
			return "", err
		}
		agg = &resourceAggregations{Global: &global}
	}

	if agg == nil || agg.Global == nil {
		return "{}", nil
	}

	formatted := formattedExecutionMetrics{
		Global: &formattedResourceStats{},
	}

	if agg.Global.CPU != nil && agg.Global.CPU.Millicores != nil {
		formatted.Global.CPUMillicores = &formattedMetricStat{
			Min: agg.Global.CPU.Millicores.Min,
			Max: agg.Global.CPU.Millicores.Max,
			Avg: agg.Global.CPU.Millicores.Avg,
		}
	}

	if agg.Global.Memory != nil && agg.Global.Memory.Used != nil {
		formatted.Global.MemoryUsedBytes = &formattedMetricStat{
			Min: agg.Global.Memory.Used.Min,
			Max: agg.Global.Memory.Used.Max,
			Avg: agg.Global.Memory.Used.Avg,
		}
	}

	if agg.Global.Network != nil {
		if agg.Global.Network.BytesRecvPerS != nil {
			formatted.Global.NetworkRecvKBps = &formattedMetricStat{
				Min: agg.Global.Network.BytesRecvPerS.Min / 1024,
				Max: agg.Global.Network.BytesRecvPerS.Max / 1024,
				Avg: agg.Global.Network.BytesRecvPerS.Avg / 1024,
			}
		}
		if agg.Global.Network.BytesSentPerS != nil {
			formatted.Global.NetworkSentKBps = &formattedMetricStat{
				Min: agg.Global.Network.BytesSentPerS.Min / 1024,
				Max: agg.Global.Network.BytesSentPerS.Max / 1024,
				Avg: agg.Global.Network.BytesSentPerS.Avg / 1024,
			}
		}
	}

	return FormatJSON(formatted)
}
