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
