package formatters

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatListWorkflows(t *testing.T) {
	// Use shared helper for empty input test cases
	RunEmptyInputCases(t, FormatListWorkflows, "[]")

	t.Run("empty JSON array returns empty array", func(t *testing.T) {
		result, err := FormatListWorkflows("[]")
		require.NoError(t, err)
		assert.Equal(t, "[]", result)
	})

	t.Run("parses JSON input with full workflow data", func(t *testing.T) {
		input := `[{
			"workflow": {
				"name": "test-workflow",
				"namespace": "testkube",
				"description": "A test workflow",
				"labels": {"team": "platform"},
				"created": "2025-01-15T10:00:00Z",
				"updated": "2025-01-20T15:30:00Z",
				"status": {
					"health": {
						"passRate": 0.95,
						"flipRate": 0.05,
						"overallHealth": 0.9025
					}
				}
			},
			"latestExecution": {
				"id": "exec-123",
				"name": "test-workflow-1",
				"number": 42,
				"scheduledAt": "2025-01-20T15:00:00Z",
				"result": {
					"status": "passed",
					"duration": "2m30s"
				}
			}
		}]`

		result, err := FormatListWorkflows(input)
		require.NoError(t, err)

		var output []formattedWorkflow
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)
		require.Len(t, output, 1)

		wf := output[0]
		assert.Equal(t, "test-workflow", wf.Name)
		assert.Equal(t, "testkube", wf.Namespace)
		assert.Equal(t, "A test workflow", wf.Description)
		assert.Equal(t, map[string]string{"team": "platform"}, wf.Labels)

		require.NotNil(t, wf.Health)
		assert.Equal(t, 0.95, wf.Health.PassRate)
		assert.Equal(t, 0.05, wf.Health.FlipRate)
		assert.Equal(t, 0.9025, wf.Health.OverallHealth)

		require.NotNil(t, wf.Latest)
		assert.Equal(t, "exec-123", wf.Latest.ID)
		assert.Equal(t, "test-workflow-1", wf.Latest.Name)
		assert.Equal(t, int32(42), wf.Latest.Number)
		assert.Equal(t, "passed", wf.Latest.Status)
		assert.Equal(t, "2m30s", wf.Latest.Duration)
	})

	t.Run("parses JSON input with minimal workflow data", func(t *testing.T) {
		input := `[{"workflow": {"name": "minimal-workflow"}}]`

		result, err := FormatListWorkflows(input)
		require.NoError(t, err)

		var output []formattedWorkflow
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)
		require.Len(t, output, 1)

		wf := output[0]
		assert.Equal(t, "minimal-workflow", wf.Name)
		assert.Nil(t, wf.Health)
		assert.Nil(t, wf.Latest)
	})

	t.Run("handles multiple workflows", func(t *testing.T) {
		input := `[
			{"workflow": {"name": "workflow-1"}},
			{"workflow": {"name": "workflow-2"}},
			{"workflow": {"name": "workflow-3"}}
		]`

		result, err := FormatListWorkflows(input)
		require.NoError(t, err)

		var output []formattedWorkflow
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)
		require.Len(t, output, 3)

		assert.Equal(t, "workflow-1", output[0].Name)
		assert.Equal(t, "workflow-2", output[1].Name)
		assert.Equal(t, "workflow-3", output[2].Name)
	})

	t.Run("preserves timestamps correctly", func(t *testing.T) {
		input := `[{
			"workflow": {
				"name": "timestamp-test",
				"created": "2025-06-15T10:30:00Z",
				"updated": "2025-06-20T14:45:00Z"
			},
			"latestExecution": {
				"id": "exec-ts",
				"name": "timestamp-test-1",
				"scheduledAt": "2025-06-20T14:00:00Z"
			}
		}]`

		result, err := FormatListWorkflows(input)
		require.NoError(t, err)

		var output []formattedWorkflow
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)
		require.Len(t, output, 1)

		wf := output[0]
		expectedCreated := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
		expectedUpdated := time.Date(2025, 6, 20, 14, 45, 0, 0, time.UTC)
		expectedScheduled := time.Date(2025, 6, 20, 14, 0, 0, 0, time.UTC)

		assert.True(t, wf.Created.Equal(expectedCreated))
		assert.True(t, wf.Updated.Equal(expectedUpdated))
		require.NotNil(t, wf.Latest)
		assert.True(t, wf.Latest.ScheduledAt.Equal(expectedScheduled))
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `[{"invalid json`
		_, err := FormatListWorkflows(input)
		assert.Error(t, err)
	})

	t.Run("handles workflow without latestExecution", func(t *testing.T) {
		input := `[{
			"workflow": {
				"name": "no-executions",
				"description": "Never ran"
			}
		}]`

		result, err := FormatListWorkflows(input)
		require.NoError(t, err)

		var output []formattedWorkflow
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)
		require.Len(t, output, 1)

		wf := output[0]
		assert.Equal(t, "no-executions", wf.Name)
		assert.Nil(t, wf.Latest)
	})

	t.Run("handles latestExecution without result", func(t *testing.T) {
		input := `[{
			"workflow": {"name": "pending-workflow"},
			"latestExecution": {
				"id": "exec-pending",
				"name": "pending-workflow-1"
			}
		}]`

		result, err := FormatListWorkflows(input)
		require.NoError(t, err)

		var output []formattedWorkflow
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)
		require.Len(t, output, 1)

		wf := output[0]
		require.NotNil(t, wf.Latest)
		assert.Equal(t, "exec-pending", wf.Latest.ID)
		assert.Empty(t, wf.Latest.Status)
		assert.Empty(t, wf.Latest.Duration)
	})

	t.Run("output is compact JSON without extra fields", func(t *testing.T) {
		// This tests that we're not including the full workflow spec or other large fields
		// Note: We use a simplified spec that doesn't include BoxedStringList types
		input := `[{
			"workflow": {
				"name": "compact-test",
				"spec": {
					"steps": [
						{"name": "step1"}
					]
				},
				"annotations": {"large": "annotation"},
				"readOnly": false
			}
		}]`

		result, err := FormatListWorkflows(input)
		require.NoError(t, err)

		// Verify spec is not in output
		assert.NotContains(t, result, "steps")
		assert.NotContains(t, result, "step1")
		assert.NotContains(t, result, "annotations")
		assert.NotContains(t, result, "readOnly")

		// But name should be there
		assert.Contains(t, result, "compact-test")
	})
}

func TestFormatGetWorkflow(t *testing.T) {
	// Use shared helper for empty input test cases
	RunEmptyInputCases(t, FormatGetWorkflow, "{}")

	t.Run("parses JSON input with full workflow data", func(t *testing.T) {
		input := `{
			"name": "k6-workflow-smoke",
			"namespace": "testkube-agent",
			"description": "Performance test workflow",
			"labels": {"tool": "k6", "category": "performance-testing"},
			"annotations": {"testkube.io/icon": "k6"},
			"created": "2025-05-19T12:42:22Z",
			"updated": "2025-05-19T12:42:22Z",
			"spec": {
				"content": {"git": {"uri": "https://github.com/kubeshop/testkube"}},
				"steps": [{"name": "Run test", "shell": "k6 run test.js"}]
			},
			"status": {
				"health": {
					"passRate": 1,
					"flipRate": 0,
					"overallHealth": 1
				}
			}
		}`

		result, err := FormatGetWorkflow(input)
		require.NoError(t, err)

		var output formattedWorkflowDetails
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Equal(t, "k6-workflow-smoke", output.Name)
		assert.Equal(t, "testkube-agent", output.Namespace)
		assert.Equal(t, "Performance test workflow", output.Description)
		assert.Equal(t, map[string]string{"tool": "k6", "category": "performance-testing"}, output.Labels)

		require.NotNil(t, output.Health)
		assert.Equal(t, 1.0, output.Health.PassRate)
		assert.Equal(t, 0.0, output.Health.FlipRate)
		assert.Equal(t, 1.0, output.Health.OverallHealth)

		// Spec should be preserved
		require.NotNil(t, output.Spec)
	})

	t.Run("strips annotations from output", func(t *testing.T) {
		input := `{
			"name": "test-workflow",
			"annotations": {"testkube.io/icon": "k6", "other": "annotation"}
		}`

		result, err := FormatGetWorkflow(input)
		require.NoError(t, err)

		assert.NotContains(t, result, "annotations")
		assert.NotContains(t, result, "testkube.io/icon")
		assert.Contains(t, result, "test-workflow")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{"invalid json`
		_, err := FormatGetWorkflow(input)
		assert.Error(t, err)
	})

	t.Run("parses TestWorkflowWithExecutionSummary format", func(t *testing.T) {
		// This is the format returned by /agent/test-workflow-with-executions/{name}
		input := `{
			"workflow": {
				"name": "k6-workflow-smoke",
				"namespace": "testkube-agent",
				"labels": {"tool": "k6"},
				"description": "Performance test",
				"spec": {"steps": [{"name": "run test", "shell": "k6 run test.js"}]}
			},
			"latestExecution": {
				"id": "abc123",
				"name": "k6-workflow-smoke-1",
				"status": "passed"
			}
		}`

		result, err := FormatGetWorkflow(input)
		require.NoError(t, err)

		var output formattedWorkflowDetails
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Equal(t, "k6-workflow-smoke", output.Name)
		assert.Equal(t, "testkube-agent", output.Namespace)
		assert.Equal(t, "Performance test", output.Description)
		assert.Equal(t, map[string]string{"tool": "k6"}, output.Labels)
		require.NotNil(t, output.Spec)
	})

	t.Run("handles wrapped workflow with health status", func(t *testing.T) {
		input := `{
			"workflow": {
				"name": "test-workflow",
				"status": {
					"health": {
						"passRate": 0.95,
						"flipRate": 0.05,
						"overallHealth": 0.9
					}
				}
			}
		}`

		result, err := FormatGetWorkflow(input)
		require.NoError(t, err)

		var output formattedWorkflowDetails
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Equal(t, "test-workflow", output.Name)
		require.NotNil(t, output.Health)
		assert.Equal(t, 0.95, output.Health.PassRate)
		assert.Equal(t, 0.05, output.Health.FlipRate)
		assert.Equal(t, 0.9, output.Health.OverallHealth)
	})
}

func TestFormatGetWorkflowDefinition(t *testing.T) {
	// Use shared helper for empty input test cases
	RunEmptyInputCases(t, FormatGetWorkflowDefinition, "{}")

	t.Run("passes through valid JSON unchanged", func(t *testing.T) {
		input := `{"name":"test-workflow","spec":{"steps":[{"name":"step1","shell":"echo hello"}]}}`

		result, err := FormatGetWorkflowDefinition(input)
		require.NoError(t, err)

		// Should return input unchanged
		assert.Equal(t, input, result)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{"invalid json`
		_, err := FormatGetWorkflowDefinition(input)
		assert.Error(t, err)
	})
}

func TestFormatRunWorkflow(t *testing.T) {
	// Use shared helper for empty input test cases
	RunEmptyInputCases(t, FormatRunWorkflow, "{}")

	t.Run("parses execution result with full data", func(t *testing.T) {
		input := `{
			"id": "697f5d90fe477e023c839c1c",
			"name": "k6-workflow-smoke-20662",
			"namespace": "testkube-agent",
			"number": 20662,
			"scheduledAt": "2026-02-01T14:05:04.845Z",
			"workflow": {"name": "k6-workflow-smoke"},
			"result": {"status": "queued"},
			"signature": [{"ref": "step1", "name": "Run test"}],
			"output": [{"name": "artifact1"}]
		}`

		result, err := FormatRunWorkflow(input)
		require.NoError(t, err)

		var output formattedRunWorkflowResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Equal(t, "697f5d90fe477e023c839c1c", output.ID)
		assert.Equal(t, "k6-workflow-smoke-20662", output.Name)
		assert.Equal(t, int32(20662), output.Number)
		assert.Equal(t, "k6-workflow-smoke", output.WorkflowName)
		assert.Equal(t, "queued", output.Status)
	})

	t.Run("strips verbose fields", func(t *testing.T) {
		input := `{
			"id": "exec-123",
			"name": "test-exec",
			"workflow": {"name": "test-wf", "spec": {"steps": []}},
			"signature": [{"ref": "step1"}],
			"output": [{"name": "artifact1"}],
			"result": {"status": "passed"}
		}`

		result, err := FormatRunWorkflow(input)
		require.NoError(t, err)

		assert.NotContains(t, result, "signature")
		assert.NotContains(t, result, "output")
		assert.NotContains(t, result, "spec")
		assert.Contains(t, result, "exec-123")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{"invalid json`
		_, err := FormatRunWorkflow(input)
		assert.Error(t, err)
	})
}

func TestFormatCreateWorkflow(t *testing.T) {
	// Use shared helper for empty input test cases
	RunEmptyInputCases(t, FormatCreateWorkflow, "{}")

	t.Run("parses created workflow response", func(t *testing.T) {
		input := `{
			"name": "new-workflow",
			"namespace": "testkube-agent",
			"description": "A new workflow",
			"labels": {"env": "test"},
			"created": "2026-02-01T10:00:00Z",
			"spec": {"steps": [{"name": "step1"}]},
			"annotations": {"key": "value"},
			"status": {"health": {"passRate": 0}}
		}`

		result, err := FormatCreateWorkflow(input)
		require.NoError(t, err)

		var output formattedCreateWorkflowResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Equal(t, "new-workflow", output.Name)
		assert.Equal(t, "testkube-agent", output.Namespace)
		assert.Equal(t, "A new workflow", output.Description)
		assert.Equal(t, map[string]string{"env": "test"}, output.Labels)
	})

	t.Run("strips spec and annotations", func(t *testing.T) {
		input := `{
			"name": "compact-workflow",
			"spec": {"steps": [{"name": "step1", "shell": "echo test"}]},
			"annotations": {"key": "value"}
		}`

		result, err := FormatCreateWorkflow(input)
		require.NoError(t, err)

		assert.NotContains(t, result, "spec")
		assert.NotContains(t, result, "steps")
		assert.NotContains(t, result, "annotations")
		assert.Contains(t, result, "compact-workflow")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{"invalid json`
		_, err := FormatCreateWorkflow(input)
		assert.Error(t, err)
	})
}

func TestFormatUpdateWorkflow(t *testing.T) {
	// Use shared helper for empty input test cases
	RunEmptyInputCases(t, FormatUpdateWorkflow, "{}")

	t.Run("uses same format as create", func(t *testing.T) {
		input := `{
			"name": "updated-workflow",
			"namespace": "testkube-agent",
			"description": "Updated description",
			"labels": {"version": "2"},
			"created": "2026-01-01T10:00:00Z"
		}`

		result, err := FormatUpdateWorkflow(input)
		require.NoError(t, err)

		var output formattedCreateWorkflowResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Equal(t, "updated-workflow", output.Name)
		assert.Equal(t, "Updated description", output.Description)
	})
}

func TestFormatGetWorkflowMetrics(t *testing.T) {
	// Use shared helper for empty input test cases
	RunEmptyInputCases(t, FormatGetWorkflowMetrics, "{}")

	t.Run("parses metrics with full data", func(t *testing.T) {
		input := `{
			"passFailRatio": 100,
			"totalExecutions": 50,
			"executionDurationP50": "36.21s",
			"executionDurationP90": "37.48s",
			"executionDurationP95": "38.05s",
			"executionDurationP99": "39.89s",
			"executions": [
				{"executionId": "exec1", "duration": "35s", "status": "passed"},
				{"executionId": "exec2", "duration": "36s", "status": "passed"}
			]
		}`

		result, err := FormatGetWorkflowMetrics(input)
		require.NoError(t, err)

		var output formattedWorkflowMetrics
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Equal(t, 100.0, output.PassFailRatio)
		assert.Equal(t, 50, output.TotalExecutions)
		assert.Equal(t, "36.21s", output.ExecutionDurationP50)
		assert.Equal(t, "37.48s", output.ExecutionDurationP90)
		assert.Equal(t, "38.05s", output.ExecutionDurationP95)
		assert.Equal(t, "39.89s", output.ExecutionDurationP99)
	})

	t.Run("strips executions array", func(t *testing.T) {
		input := `{
			"passFailRatio": 100,
			"totalExecutions": 10,
			"executionDurationP50": "30s",
			"executionDurationP90": "35s",
			"executionDurationP95": "36s",
			"executionDurationP99": "40s",
			"executions": [
				{"executionId": "exec1", "name": "test-1", "duration": "35s"},
				{"executionId": "exec2", "name": "test-2", "duration": "36s"}
			]
		}`

		result, err := FormatGetWorkflowMetrics(input)
		require.NoError(t, err)

		assert.NotContains(t, result, "exec1")
		assert.NotContains(t, result, "exec2")
		assert.NotContains(t, result, "test-1")
		assert.NotContains(t, result, "test-2")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{"invalid json`
		_, err := FormatGetWorkflowMetrics(input)
		assert.Error(t, err)
	})
}

func TestFormatGetWorkflowExecutionMetrics(t *testing.T) {
	// Use shared helper for empty input test cases (wrap to match expected signature)
	RunEmptyInputCases(t, func(s string) (string, error) {
		return FormatGetWorkflowExecutionMetrics(s, 0)
	}, "{}")

	t.Run("parses AggregatedMetrics with time-series data", func(t *testing.T) {
		input := `{
			"workflow": "test-workflow",
			"execution": "exec-123",
			"result": {
				"startedAt": "2025-01-20T15:00:00Z",
				"finishedAt": "2025-01-20T15:02:30Z"
			},
			"metrics": [
				{
					"step": "run-tests",
					"tags": {},
					"data": [
						{
							"measurement": "cpu",
							"fields": "millicores",
							"values": [[1706000000000, 25.5], [1706000001000, 50.0], [1706000002000, 75.5]]
						},
						{
							"measurement": "memory",
							"fields": "used",
							"values": [[1706000000000, 87801856], [1706000001000, 90000000], [1706000002000, 92520448]]
						}
					]
				}
			]
		}`

		result, err := FormatGetWorkflowExecutionMetrics(input, 0)
		require.NoError(t, err)

		var output formattedExecutionMetrics
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Equal(t, "test-workflow", output.Workflow)
		assert.Equal(t, "exec-123", output.Execution)
		require.NotNil(t, output.Result)
		assert.Equal(t, "2025-01-20T15:00:00Z", output.Result.StartedAt)
		assert.Equal(t, "2025-01-20T15:02:30Z", output.Result.FinishedAt)

		require.Len(t, output.Steps, 1)
		assert.Equal(t, "run-tests", output.Steps[0].Step)
		require.Len(t, output.Steps[0].Series, 2)

		// CPU series
		cpuSeries := output.Steps[0].Series[0]
		assert.Equal(t, "cpu.millicores", cpuSeries.Metric)
		assert.Equal(t, 3, cpuSeries.SampleCount)
		assert.Equal(t, 25.5, cpuSeries.Summary.Min)
		assert.Equal(t, 75.5, cpuSeries.Summary.Max)
		assert.Equal(t, 50.33, cpuSeries.Summary.Avg)
		assert.Len(t, cpuSeries.Samples, 3) // fewer than default, all kept

		// Memory series
		memSeries := output.Steps[0].Series[1]
		assert.Equal(t, "memory.used", memSeries.Metric)
		assert.Equal(t, 3, memSeries.SampleCount)
		assert.Equal(t, 87801856.0, memSeries.Summary.Min)
		assert.Equal(t, 92520448.0, memSeries.Summary.Max)
	})

	t.Run("preserves multiple steps", func(t *testing.T) {
		input := `{
			"workflow": "multi-step",
			"execution": "exec-456",
			"metrics": [
				{
					"step": "step1",
					"data": [
						{"measurement": "cpu", "fields": "millicores", "values": [[1000, 10], [2000, 20]]}
					]
				},
				{
					"step": "step2",
					"data": [
						{"measurement": "cpu", "fields": "millicores", "values": [[3000, 30], [4000, 40]]}
					]
				}
			]
		}`

		result, err := FormatGetWorkflowExecutionMetrics(input, 0)
		require.NoError(t, err)

		var output formattedExecutionMetrics
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		require.Len(t, output.Steps, 2)
		assert.Equal(t, "step1", output.Steps[0].Step)
		assert.Equal(t, "step2", output.Steps[1].Step)
		assert.Equal(t, 10.0, output.Steps[0].Series[0].Summary.Min)
		assert.Equal(t, 30.0, output.Steps[1].Series[0].Summary.Min)
	})

	t.Run("downsamples large time-series with default", func(t *testing.T) {
		// Build a series with 200 data points (more than defaultMaxSamplesPerSeries)
		values := make([][2]float64, 200)
		for i := 0; i < 200; i++ {
			values[i] = [2]float64{float64(1000 + i*1000), float64(i)}
		}
		valuesJSON, _ := json.Marshal(values)

		input := `{
			"workflow": "big-run",
			"execution": "exec-789",
			"metrics": [
				{
					"step": "step1",
					"data": [
						{"measurement": "cpu", "fields": "millicores", "values": ` + string(valuesJSON) + `}
					]
				}
			]
		}`

		result, err := FormatGetWorkflowExecutionMetrics(input, 0) // 0 = use default
		require.NoError(t, err)

		var output formattedExecutionMetrics
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		series := output.Steps[0].Series[0]
		assert.Equal(t, 200, series.SampleCount)                              // original count preserved
		assert.LessOrEqual(t, len(series.Samples), defaultMaxSamplesPerSeries) // downsampled to default
		assert.Equal(t, values[0], series.Samples[0])                          // first point kept
		assert.Equal(t, values[199], series.Samples[len(series.Samples)-1])    // last point kept
	})

	t.Run("respects custom maxSamples parameter", func(t *testing.T) {
		// Build a series with 100 data points
		values := make([][2]float64, 100)
		for i := 0; i < 100; i++ {
			values[i] = [2]float64{float64(1000 + i*1000), float64(i)}
		}
		valuesJSON, _ := json.Marshal(values)

		input := `{
			"workflow": "custom-run",
			"execution": "exec-custom",
			"metrics": [
				{
					"step": "step1",
					"data": [
						{"measurement": "cpu", "fields": "millicores", "values": ` + string(valuesJSON) + `}
					]
				}
			]
		}`

		// Request only 10 samples
		result, err := FormatGetWorkflowExecutionMetrics(input, 10)
		require.NoError(t, err)

		var output formattedExecutionMetrics
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		series := output.Steps[0].Series[0]
		assert.Equal(t, 100, series.SampleCount)    // original count preserved
		assert.Equal(t, 10, len(series.Samples))     // downsampled to requested 10
		assert.Equal(t, values[0], series.Samples[0]) // first point kept
		assert.Equal(t, values[99], series.Samples[9]) // last point kept

		// Request 200 samples (more than available) — should return all
		result, err = FormatGetWorkflowExecutionMetrics(input, 200)
		require.NoError(t, err)

		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)
		assert.Equal(t, 100, len(output.Steps[0].Series[0].Samples)) // all points returned
	})

	t.Run("returns empty for non-AggregatedMetrics input", func(t *testing.T) {
		input := `{"someOtherField": "value"}`
		result, err := FormatGetWorkflowExecutionMetrics(input, 0)
		require.NoError(t, err)
		assert.Equal(t, "{}", result)
	})

	t.Run("handles empty metrics array", func(t *testing.T) {
		input := `{"workflow": "test", "execution": "exec-1", "metrics": []}`
		result, err := FormatGetWorkflowExecutionMetrics(input, 0)
		require.NoError(t, err)

		var output formattedExecutionMetrics
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)
		assert.Equal(t, "test", output.Workflow)
		assert.Empty(t, output.Steps)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{"invalid json`
		_, err := FormatGetWorkflowExecutionMetrics(input, 0)
		assert.Error(t, err)
	})
}
