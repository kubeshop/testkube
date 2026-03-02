package formatters

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatListExecutions(t *testing.T) {
	// Use shared helper for empty input test cases
	RunEmptyInputCases(t, FormatListExecutions, "{}")

	t.Run("parses JSON input with full execution data", func(t *testing.T) {
		input := `{
			"totals": {"results": 100, "passed": 80, "failed": 15, "queued": 3, "running": 2},
			"filtered": {"results": 10, "passed": 8, "failed": 2, "queued": 0, "running": 0},
			"results": [{
				"id": "exec-123",
				"name": "test-workflow-42",
				"number": 42,
				"scheduledAt": "2025-01-20T15:00:00Z",
				"workflow": {"name": "test-workflow"},
				"result": {
					"status": "passed",
					"duration": "2m30s"
				},
				"runningContext": {
					"interface": {"type": "cli"},
					"actor": {
						"name": "john.doe@example.com",
						"email": "john.doe@example.com",
						"type": "user"
					}
				},
				"configParams": {"param1": {"value": "val1"}},
				"resourceAggregations": {"cpu": {"avg": 100}}
			}]
		}`

		result, err := FormatListExecutions(input)
		require.NoError(t, err)

		var output formattedExecutionsResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		// Verify totals are preserved
		require.NotNil(t, output.Totals)
		assert.Equal(t, int32(100), output.Totals.Results)
		assert.Equal(t, int32(80), output.Totals.Passed)

		// Verify filtered is preserved
		require.NotNil(t, output.Filtered)
		assert.Equal(t, int32(10), output.Filtered.Results)

		// Verify execution data
		require.Len(t, output.Results, 1)
		exec := output.Results[0]
		assert.Equal(t, "exec-123", exec.ID)
		assert.Equal(t, "test-workflow-42", exec.Name)
		assert.Equal(t, int32(42), exec.Number)
		assert.Equal(t, "test-workflow", exec.WorkflowName)
		assert.Equal(t, "passed", exec.Status)
		assert.Equal(t, "2m30s", exec.Duration)
		assert.Equal(t, "john.doe@example.com", exec.ActorName)
		assert.Equal(t, "user", exec.ActorType)

		// Verify stripped fields are not in output
		assert.NotContains(t, result, "configParams")
		assert.NotContains(t, result, "resourceAggregations")
		assert.NotContains(t, result, "interface")
	})

	t.Run("parses JSON input with minimal execution data", func(t *testing.T) {
		input := `{
			"totals": {"results": 1},
			"results": [{"id": "exec-minimal", "name": "minimal-1"}]
		}`

		result, err := FormatListExecutions(input)
		require.NoError(t, err)

		var output formattedExecutionsResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		require.Len(t, output.Results, 1)
		exec := output.Results[0]
		assert.Equal(t, "exec-minimal", exec.ID)
		assert.Equal(t, "minimal-1", exec.Name)
		assert.Empty(t, exec.WorkflowName)
		assert.Empty(t, exec.Status)
		assert.Empty(t, exec.ActorName)
	})

	t.Run("handles empty results array", func(t *testing.T) {
		input := `{
			"totals": {"results": 0},
			"filtered": {"results": 0},
			"results": []
		}`

		result, err := FormatListExecutions(input)
		require.NoError(t, err)

		var output formattedExecutionsResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Empty(t, output.Results)
		require.NotNil(t, output.Totals)
		assert.Equal(t, int32(0), output.Totals.Results)
	})

	t.Run("handles multiple executions", func(t *testing.T) {
		input := `{
			"totals": {"results": 3},
			"results": [
				{"id": "exec-1", "name": "wf-1", "result": {"status": "passed"}},
				{"id": "exec-2", "name": "wf-2", "result": {"status": "failed"}},
				{"id": "exec-3", "name": "wf-3", "result": {"status": "running"}}
			]
		}`

		result, err := FormatListExecutions(input)
		require.NoError(t, err)

		var output formattedExecutionsResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		require.Len(t, output.Results, 3)
		assert.Equal(t, "passed", output.Results[0].Status)
		assert.Equal(t, "failed", output.Results[1].Status)
		assert.Equal(t, "running", output.Results[2].Status)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{"invalid json`
		_, err := FormatListExecutions(input)
		assert.Error(t, err)
	})
}

func TestFormatExecutionInfo(t *testing.T) {
	RunEmptyInputCases(t, FormatExecutionInfo, "{}")

	t.Run("parses JSON input with full execution data", func(t *testing.T) {
		input := `{
			"id": "6749b3d148ee32f39df3fc5a",
			"name": "distributed-k6-4",
			"number": 4,
			"namespace": "testkube-agent",
			"scheduledAt": "2024-11-29T12:30:09.465Z",
			"signature": [
				{"ref": "rr4m7s5", "category": "Clone Git repository"},
				{"ref": "rvb2k9b", "name": "Run test", "category": "Run in parallel"}
			],
			"result": {
				"status": "passed",
				"predictedStatus": "passed",
				"queuedAt": "2024-11-29T12:30:09.465Z",
				"startedAt": "2024-11-29T12:30:11Z",
				"finishedAt": "2024-11-29T12:30:38.206Z",
				"duration": "28.741s",
				"totalDuration": "28.741s",
				"steps": {
					"rr4m7s5": {"status": "passed", "startedAt": "2024-11-29T12:30:12.364Z"}
				}
			},
			"output": [
				{"ref": "tktw-init", "name": "pod", "value": {"nodeName": "node-1"}}
			],
			"workflow": {
				"name": "distributed-k6",
				"namespace": "testkube-agent",
				"spec": {"config": {"workers": {"type": "integer"}}}
			},
			"resolvedWorkflow": {
				"name": "distributed-k6",
				"spec": {"steps": [{"name": "Run test"}]}
			},
			"runningContext": {
				"interface": {"type": "ui"},
				"actor": {"name": "john.doe@example.com", "type": "user"}
			},
			"configParams": {
				"duration": {"value": "10s"},
				"vus": {"defaultValue": "5"}
			},
			"tags": {"env": "test"},
			"resourceAggregations": {"cpu": {"avg": 100}}
		}`

		result, err := FormatExecutionInfo(input)
		require.NoError(t, err)

		var output formattedExecutionInfo
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		// Verify essential fields are preserved
		assert.Equal(t, "6749b3d148ee32f39df3fc5a", output.ID)
		assert.Equal(t, "distributed-k6-4", output.Name)
		assert.Equal(t, int32(4), output.Number)
		assert.Equal(t, "testkube-agent", output.Namespace)
		assert.Equal(t, "distributed-k6", output.WorkflowName)

		// Verify result is extracted
		require.NotNil(t, output.Result)
		assert.Equal(t, "passed", output.Result.Status)
		assert.Equal(t, "passed", output.Result.PredictedStatus)
		assert.Equal(t, "28.741s", output.Result.Duration)

		// Verify signature is extracted
		require.Len(t, output.Signature, 2)
		assert.Equal(t, "rr4m7s5", output.Signature[0].Ref)
		assert.Equal(t, "Clone Git repository", output.Signature[0].Category)
		assert.Equal(t, "Run test", output.Signature[1].Name)

		// Verify config params are simplified
		assert.Equal(t, "10s", output.ConfigParams["duration"])
		assert.Equal(t, "5", output.ConfigParams["vus"])

		// Verify actor info
		assert.Equal(t, "john.doe@example.com", output.ActorName)
		assert.Equal(t, "user", output.ActorType)

		// Verify tags preserved
		assert.Equal(t, "test", output.Tags["env"])

		// Verify stripped fields are not in output
		assert.NotContains(t, result, "resolvedWorkflow")
		assert.NotContains(t, result, "resourceAggregations")
		assert.NotContains(t, result, "output")
		assert.NotContains(t, result, "steps")
		assert.NotContains(t, result, "spec")
	})

	t.Run("handles minimal execution data", func(t *testing.T) {
		input := `{"id": "exec-123", "name": "test-1"}`

		result, err := FormatExecutionInfo(input)
		require.NoError(t, err)

		var output formattedExecutionInfo
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Equal(t, "exec-123", output.ID)
		assert.Equal(t, "test-1", output.Name)
		assert.Nil(t, output.Result)
		assert.Empty(t, output.Signature)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{"invalid json`
		_, err := FormatExecutionInfo(input)
		assert.Error(t, err)
	})
}

func TestFormatAbortExecution(t *testing.T) {
	RunEmptyInputCases(t, FormatAbortExecution, "{}")

	t.Run("parses abort response with full data", func(t *testing.T) {
		input := `{
			"id": "exec-aborted-123",
			"name": "workflow-42",
			"number": 42,
			"result": {
				"status": "aborted",
				"duration": "5s",
				"steps": {"step1": {"status": "aborted"}}
			},
			"workflow": {"name": "workflow", "spec": {}},
			"output": [{"ref": "step1", "value": {}}]
		}`

		result, err := FormatAbortExecution(input)
		require.NoError(t, err)

		var output formattedAbortResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Equal(t, "exec-aborted-123", output.ID)
		assert.Equal(t, "workflow-42", output.Name)
		assert.Equal(t, "aborted", output.Status)

		// Verify stripped fields are not in output (check for JSON keys)
		assert.NotContains(t, result, `"workflow":`)
		assert.NotContains(t, result, `"output":`)
		assert.NotContains(t, result, `"steps":`)
		assert.NotContains(t, result, `"duration":`)
	})

	t.Run("handles minimal response", func(t *testing.T) {
		input := `{"id": "exec-1", "name": "wf-1"}`

		result, err := FormatAbortExecution(input)
		require.NoError(t, err)

		var output formattedAbortResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Equal(t, "exec-1", output.ID)
		assert.Equal(t, "wf-1", output.Name)
		assert.Empty(t, output.Status)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `{"invalid json`
		_, err := FormatAbortExecution(input)
		assert.Error(t, err)
	})
}

func TestFormatWaitForExecutions(t *testing.T) {
	RunEmptyInputCases(t, FormatWaitForExecutions, "[]")

	t.Run("parses array of execution results", func(t *testing.T) {
		input := `[
			{
				"id": "exec-1",
				"name": "workflow-1",
				"result": {"status": "passed", "duration": "10s"},
				"workflow": {"name": "wf", "spec": {}},
				"output": []
			},
			{
				"id": "exec-2",
				"name": "workflow-2",
				"result": {"status": "failed", "duration": "5s"}
			},
			{
				"id": "exec-3",
				"name": "workflow-3",
				"result": {"status": "aborted"}
			}
		]`

		result, err := FormatWaitForExecutions(input)
		require.NoError(t, err)

		var output formattedWaitResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		require.Len(t, output.Executions, 3)

		assert.Equal(t, "exec-1", output.Executions[0].ID)
		assert.Equal(t, "workflow-1", output.Executions[0].Name)
		assert.Equal(t, "passed", output.Executions[0].Status)
		assert.Equal(t, "10s", output.Executions[0].Duration)

		assert.Equal(t, "exec-2", output.Executions[1].ID)
		assert.Equal(t, "failed", output.Executions[1].Status)

		assert.Equal(t, "exec-3", output.Executions[2].ID)
		assert.Equal(t, "aborted", output.Executions[2].Status)
		assert.Empty(t, output.Executions[2].Duration)

		// Verify stripped fields are not in output (check for JSON keys)
		assert.NotContains(t, result, `"workflow":`)
		assert.NotContains(t, result, `"output":`)
		assert.NotContains(t, result, `"spec":`)
	})

	t.Run("handles empty array", func(t *testing.T) {
		input := `[]`

		result, err := FormatWaitForExecutions(input)
		require.NoError(t, err)

		var output formattedWaitResult
		err = json.Unmarshal([]byte(result), &output)
		require.NoError(t, err)

		assert.Empty(t, output.Executions)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := `[{"invalid json`
		_, err := FormatWaitForExecutions(input)
		assert.Error(t, err)
	})
}
