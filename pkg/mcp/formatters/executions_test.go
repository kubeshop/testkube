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
