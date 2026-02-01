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
	// Use shared helper for empty input test cases
	RunEmptyInputCases(t, FormatGetWorkflowExecutionMetrics, "{}")

	t.Run("parses execution metrics from resourceAggregations", func(t *testing.T) {
		input := `{
			"resourceAggregations": {
				"global": {
					"cpu": {
						"millicores": {"min": 16, "max": 112, "avg": 27.9, "total": 837}
					},
					"memory": {
						"used": {"min": 87801856, "max": 92520448, "avg": 90460023.47}
					},
					"network": {
						"bytes_recv_per_s": {"min": 132, "max": 9044, "avg": 8417.07},
						"bytes_sent_per_s": {"min": 102, "max": 579, "avg": 323.17}
					}
				},
				"step": [{"ref": "step1", "aggregations": {}}]
			}
		}`

		result, err := FormatGetWorkflowExecutionMetrics(input)
		require.NoError(t, err)

		var output formattedExecutionMetrics
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		require.NotNil(t, output.Global)
		require.NotNil(t, output.Global.CPUMillicores)
		assert.Equal(t, 16.0, output.Global.CPUMillicores.Min)
		assert.Equal(t, 112.0, output.Global.CPUMillicores.Max)
		assert.Equal(t, 27.9, output.Global.CPUMillicores.Avg)

		require.NotNil(t, output.Global.MemoryUsedBytes)
		assert.Equal(t, 87801856.0, output.Global.MemoryUsedBytes.Min)
	})

	t.Run("parses direct global structure", func(t *testing.T) {
		input := `{
			"global": {
				"cpu": {
					"millicores": {"min": 10, "max": 100, "avg": 50}
				},
				"memory": {
					"used": {"min": 1000000, "max": 2000000, "avg": 1500000}
				}
			}
		}`

		result, err := FormatGetWorkflowExecutionMetrics(input)
		require.NoError(t, err)

		var output formattedExecutionMetrics
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		require.NotNil(t, output.Global)
		require.NotNil(t, output.Global.CPUMillicores)
		assert.Equal(t, 50.0, output.Global.CPUMillicores.Avg)
	})

	t.Run("strips step-level metrics", func(t *testing.T) {
		input := `{
			"resourceAggregations": {
				"global": {
					"cpu": {"millicores": {"min": 10, "max": 100, "avg": 50}}
				},
				"step": [
					{"ref": "step1", "aggregations": {"cpu": {"millicores": {"min": 5}}}},
					{"ref": "step2", "aggregations": {"cpu": {"millicores": {"min": 5}}}}
				]
			}
		}`

		result, err := FormatGetWorkflowExecutionMetrics(input)
		require.NoError(t, err)

		assert.NotContains(t, result, "step1")
		assert.NotContains(t, result, "step2")
		assert.NotContains(t, result, "ref")
	})

	t.Run("converts network bytes to KB", func(t *testing.T) {
		input := `{
			"global": {
				"network": {
					"bytes_recv_per_s": {"min": 1024, "max": 10240, "avg": 5120},
					"bytes_sent_per_s": {"min": 512, "max": 2048, "avg": 1024}
				}
			}
		}`

		result, err := FormatGetWorkflowExecutionMetrics(input)
		require.NoError(t, err)

		var output formattedExecutionMetrics
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		require.NotNil(t, output.Global.NetworkRecvKBps)
		assert.Equal(t, 1.0, output.Global.NetworkRecvKBps.Min)  // 1024/1024 = 1
		assert.Equal(t, 10.0, output.Global.NetworkRecvKBps.Max) // 10240/1024 = 10
		assert.Equal(t, 5.0, output.Global.NetworkRecvKBps.Avg)  // 5120/1024 = 5
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{"invalid json`
		_, err := FormatGetWorkflowExecutionMetrics(input)
		assert.Error(t, err)
	})
}
