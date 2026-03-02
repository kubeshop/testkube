package formatters

import (
	"encoding/json"
	"math"
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

	return FormatJSON(formattedWorkflowMetrics(metrics))
}

// defaultMaxSamplesPerSeries is the default maximum number of time-series data points per metric.
// If a series has more samples, it will be evenly downsampled (always keeping first and last).
const defaultMaxSamplesPerSeries = 50

// formattedExecutionMetrics is a compact representation of execution resource metrics time-series.
type formattedExecutionMetrics struct {
	Workflow  string              `json:"workflow,omitempty"`
	Execution string              `json:"execution,omitempty"`
	Message   string              `json:"message,omitempty"`
	Result    *formattedTimeRange `json:"result,omitempty"`
	Steps     []formattedStepData `json:"steps,omitempty"`
}

type formattedTimeRange struct {
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

type formattedStepData struct {
	Step   string                  `json:"step"`
	Series []formattedMetricSeries `json:"series"`
}

type formattedMetricSeries struct {
	Metric      string       `json:"metric"` // e.g. "cpu.millicores", "memory.used"
	SampleCount int          `json:"sampleCount"`
	Summary     metricStats  `json:"summary"`
	Samples     [][2]float64 `json:"samples"` // [timestamp_ms, value]
}

type metricStats struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Avg float64 `json:"avg"`
}

// aggregatedMetricsInput is the shape returned by the /metrics endpoint.
type aggregatedMetricsInput struct {
	Workflow  string `json:"workflow"`
	Execution string `json:"execution"`
	Result    *struct {
		StartedAt  string `json:"startedAt"`
		FinishedAt string `json:"finishedAt"`
	} `json:"result"`
	Metrics []linkedMetricInput `json:"metrics"`
}

type linkedMetricInput struct {
	Step string           `json:"step"`
	Data []dataPointInput `json:"data"`
}

type dataPointInput struct {
	Measurement string       `json:"measurement"`
	Field       string       `json:"fields"` // note: JSON key is "fields" (singular value despite plural name)
	Values      [][2]float64 `json:"values"` // [timestamp_ms, value]
}

// FormatGetWorkflowExecutionMetrics formats raw /metrics API output for the AI agent.
// Returns compact JSON with per-step metric series, downsampled to maxSamples points (0 = default 50).
func FormatGetWorkflowExecutionMetrics(raw string, maxSamples int) (string, error) {
	if IsEmptyInput(raw) {
		return "{}", nil
	}

	if maxSamples <= 0 {
		maxSamples = defaultMaxSamplesPerSeries
	}

	var input aggregatedMetricsInput
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return "", err
	}

	// Unrecognisable response — neither workflow nor metrics present.
	if input.Workflow == "" && len(input.Metrics) == 0 {
		return "{}", nil
	}

	// Workflow is known but no metrics were collected — return an explicit message
	// rather than a bare stub so the agent can communicate this clearly.
	if len(input.Metrics) == 0 {
		return FormatJSON(formattedExecutionMetrics{
			Workflow:  input.Workflow,
			Execution: input.Execution,
			Message:   "No resource metric data was collected for this execution. This typically means the agent did not emit telemetry during the run (e.g. the execution was too short, or metric collection is not enabled for this workflow).",
		})
	}

	formatted := formattedExecutionMetrics{
		Workflow:  input.Workflow,
		Execution: input.Execution,
	}

	if input.Result != nil {
		formatted.Result = &formattedTimeRange{
			StartedAt:  input.Result.StartedAt,
			FinishedAt: input.Result.FinishedAt,
		}
	}

	for _, m := range input.Metrics {
		step := formattedStepData{
			Step: m.Step,
		}

		for _, dp := range m.Data {
			metricName := dp.Measurement
			if dp.Field != "" {
				metricName += "." + dp.Field
			}

			series := formattedMetricSeries{
				Metric:      metricName,
				SampleCount: len(dp.Values),
				Summary:     computeStats(dp.Values),
				Samples:     downsample(dp.Values, maxSamples),
			}

			step.Series = append(step.Series, series)
		}

		formatted.Steps = append(formatted.Steps, step)
	}

	return FormatJSON(formatted)
}

// computeStats calculates min/max/avg from a time-series of [timestamp, value] pairs.
func computeStats(values [][2]float64) metricStats {
	if len(values) == 0 {
		return metricStats{}
	}

	minVal := values[0][1]
	maxVal := values[0][1]
	sum := 0.0

	for _, v := range values {
		val := v[1]
		if val < minVal {
			minVal = val
		}
		if val > maxVal {
			maxVal = val
		}
		sum += val
	}

	return metricStats{
		Min: math.Round(minVal*100) / 100,
		Max: math.Round(maxVal*100) / 100,
		Avg: math.Round(sum/float64(len(values))*100) / 100,
	}
}

// downsample reduces a time-series to at most maxPoints, evenly spaced.
// Always preserves the first and last data points.
func downsample(values [][2]float64, maxPoints int) [][2]float64 {
	if len(values) <= maxPoints {
		return values
	}

	result := make([][2]float64, 0, maxPoints)
	result = append(result, values[0])

	// Evenly pick interior points
	step := float64(len(values)-1) / float64(maxPoints-1)
	for i := 1; i < maxPoints-1; i++ {
		idx := int(math.Round(float64(i) * step))
		result = append(result, values[idx])
	}

	result = append(result, values[len(values)-1])
	return result
}
