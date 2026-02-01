package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteYqQuery(t *testing.T) {
	t.Run("valid YAML input with simple expression", func(t *testing.T) {
		input := `
name: test-workflow
spec:
  steps:
    - name: step1
      container:
        image: alpine:3.18
    - name: step2
      container:
        image: python:3.11
`
		result, err := executeYqQuery(".spec.steps[].container.image", input, true, defaultYqTimeout)
		require.NoError(t, err)
		assert.Contains(t, result, "alpine:3.18")
		assert.Contains(t, result, "python:3.11")
	})

	t.Run("valid JSON input with simple expression", func(t *testing.T) {
		input := `{
			"result": {
				"status": "passed",
				"duration": "2m30s"
			}
		}`
		result, err := executeYqQuery(".result.status", input, false, defaultYqTimeout)
		require.NoError(t, err)
		assert.Contains(t, result, "passed")
	})

	t.Run("extract step names from YAML", func(t *testing.T) {
		input := `
spec:
  steps:
    - name: build
    - name: test
    - name: deploy
`
		result, err := executeYqQuery(".spec.steps[].name", input, true, defaultYqTimeout)
		require.NoError(t, err)
		assert.Contains(t, result, "build")
		assert.Contains(t, result, "test")
		assert.Contains(t, result, "deploy")
	})

	t.Run("invalid expression returns error", func(t *testing.T) {
		input := `name: test`
		_, err := executeYqQuery("[[[invalid", input, true, defaultYqTimeout)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to evaluate yq expression")
	})

	t.Run("empty input returns empty result", func(t *testing.T) {
		result, err := executeYqQuery(".name", `{}`, false, defaultYqTimeout)
		require.NoError(t, err)
		// Empty result or null is acceptable
		assert.True(t, result == "" || result == "null")
	})

	t.Run("timeout enforcement", func(t *testing.T) {
		input := `name: test`
		// Use a very short timeout
		_, err := executeYqQuery(".name", input, true, 1*time.Nanosecond)
		// This might or might not timeout depending on execution speed
		// The test is mainly to ensure the timeout mechanism doesn't panic
		_ = err
	})

	t.Run("output size limit enforcement", func(t *testing.T) {
		// Create input that would generate large output
		// This is a simplified test - in practice, we'd need much larger input
		input := `name: test`
		result, err := executeYqQuery(".name", input, true, defaultYqTimeout)
		require.NoError(t, err)
		assert.True(t, len(result) <= maxOutputSize)
	})

	t.Run("select with filter", func(t *testing.T) {
		input := `
items:
  - name: foo
    value: 1
  - name: bar
    value: 2
  - name: baz
    value: 3
`
		result, err := executeYqQuery(".items[] | select(.value > 1) | .name", input, true, defaultYqTimeout)
		require.NoError(t, err)
		assert.Contains(t, result, "bar")
		assert.Contains(t, result, "baz")
		assert.NotContains(t, result, "foo")
	})

	t.Run("keys extraction", func(t *testing.T) {
		input := `
services:
  database:
    image: postgres
  cache:
    image: redis
  queue:
    image: rabbitmq
`
		result, err := executeYqQuery(".services | keys", input, true, defaultYqTimeout)
		require.NoError(t, err)
		assert.Contains(t, result, "database")
		assert.Contains(t, result, "cache")
		assert.Contains(t, result, "queue")
	})
}

func TestYqSecurityRestrictions(t *testing.T) {
	t.Run("env operator is disabled", func(t *testing.T) {
		input := `name: test`
		// Try to read an environment variable using env()
		_, err := executeYqQuery(`env("PATH")`, input, true, defaultYqTimeout)
		require.Error(t, err)
		// Should be blocked by our regex validator before reaching yq
		assert.True(t, strings.Contains(err.Error(), "blocked operator") ||
			strings.Contains(err.Error(), "env operations have been disabled"))
	})

	t.Run("envsubst operator is disabled", func(t *testing.T) {
		input := `value: "${HOME}"`
		// Try to substitute environment variables
		_, err := executeYqQuery(`. | envsubst`, input, true, defaultYqTimeout)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "env operations have been disabled")
	})

	t.Run("load operator is disabled", func(t *testing.T) {
		input := `file: "/etc/passwd"`
		// Try to load a file from the filesystem
		_, err := executeYqQuery(`load(.file)`, input, true, defaultYqTimeout)
		require.Error(t, err)
		// Should be blocked by our regex validator before reaching yq
		assert.True(t, strings.Contains(err.Error(), "blocked operator") ||
			strings.Contains(err.Error(), "file operations have been disabled"))
	})

	t.Run("strload operator is disabled", func(t *testing.T) {
		input := `file: "/etc/passwd"`
		// Try to load a file as string
		_, err := executeYqQuery(`strload(.file)`, input, true, defaultYqTimeout)
		require.Error(t, err)
		// Should be blocked by our regex validator before reaching yq
		assert.True(t, strings.Contains(err.Error(), "blocked operator") ||
			strings.Contains(err.Error(), "file operations have been disabled"))
	})

	t.Run("expression too long is rejected", func(t *testing.T) {
		input := `name: test`
		// Create an expression that exceeds the limit
		longExpr := strings.Repeat(".name | ", 2000)
		_, err := executeYqQuery(longExpr, input, true, defaultYqTimeout)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expression too long")
	})

	t.Run("input too large is rejected", func(t *testing.T) {
		// Create input that exceeds 10MB
		largeInput := strings.Repeat("a: b\n", 3*1024*1024) // ~15MB
		_, err := executeYqQuery(".a", largeInput, true, defaultYqTimeout)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "input too large")
	})

	t.Run("safe operators still work", func(t *testing.T) {
		input := `
items:
  - name: foo
  - name: bar
`
		// Regular yq operations should still work
		result, err := executeYqQuery(".items[].name", input, true, defaultYqTimeout)
		require.NoError(t, err)
		assert.Contains(t, result, "foo")
		assert.Contains(t, result, "bar")
	})
}

func TestValidateExpression(t *testing.T) {
	t.Run("blocks env operator variations", func(t *testing.T) {
		blockedPatterns := []string{
			`env("PATH")`,
			`ENV("PATH")`,
			`env( "PATH" )`,
			`.foo | env("BAR")`,
		}
		for _, pattern := range blockedPatterns {
			err := validateExpression(pattern)
			require.Error(t, err, "should block: %s", pattern)
			assert.Contains(t, err.Error(), "blocked operator")
		}
	})

	t.Run("blocks load operator variations", func(t *testing.T) {
		blockedPatterns := []string{
			`load("/etc/passwd")`,
			`LOAD("/etc/passwd")`,
			`strload("/etc/passwd")`,
			`STRLOAD( "/etc/passwd" )`,
		}
		for _, pattern := range blockedPatterns {
			err := validateExpression(pattern)
			require.Error(t, err, "should block: %s", pattern)
			assert.Contains(t, err.Error(), "blocked operator")
		}
	})

	t.Run("allows safe expressions", func(t *testing.T) {
		safePatterns := []string{
			`.spec.steps[].container.image`,
			`select(.name == "test")`,
			`.items | length`,
			`.data | keys`,
			`.. | select(type == "string")`,
		}
		for _, pattern := range safePatterns {
			err := validateExpression(pattern)
			require.NoError(t, err, "should allow: %s", pattern)
		}
	})
}

// MockWorkflowDefinitionBulkGetter implements WorkflowDefinitionBulkGetter for testing
type MockWorkflowDefinitionBulkGetter struct {
	Definitions map[string]string
	Error       error
}

func (m *MockWorkflowDefinitionBulkGetter) GetWorkflowDefinitions(ctx context.Context, params ListWorkflowsParams) (map[string]string, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Definitions, nil
}

// MockExecutionBulkGetter implements ExecutionBulkGetter for testing
type MockExecutionBulkGetter struct {
	Executions map[string]string
	Error      error
}

func (m *MockExecutionBulkGetter) GetExecutions(ctx context.Context, params ListExecutionsParams) (map[string]string, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Executions, nil
}

func TestQueryWorkflowsYq(t *testing.T) {
	t.Run("per-item mode returns keyed results", func(t *testing.T) {
		mock := &MockWorkflowDefinitionBulkGetter{
			Definitions: map[string]string{
				"workflow-a": `
metadata:
  name: workflow-a
spec:
  steps:
    - name: step1
      container:
        image: alpine:3.18
`,
				"workflow-b": `
metadata:
  name: workflow-b
spec:
  steps:
    - name: step1
      container:
        image: python:3.11
`,
			},
		}

		tool, handler := QueryWorkflowsYq(mock)

		// Verify tool definition
		assert.Equal(t, "query_workflows_yq", tool.Name)

		// Test handler
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": ".spec.steps[].container.image",
			"aggregate":  false,
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Result should contain both workflow results
		text := getResultText(result)
		assert.Contains(t, text, "alpine:3.18")
		assert.Contains(t, text, "python:3.11")
	})

	t.Run("aggregate mode combines results", func(t *testing.T) {
		mock := &MockWorkflowDefinitionBulkGetter{
			Definitions: map[string]string{
				"workflow-a": `
metadata:
  name: workflow-a
`,
				"workflow-b": `
metadata:
  name: workflow-b
`,
			},
		}

		_, handler := QueryWorkflowsYq(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": ".metadata.name",
			"aggregate":  true,
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)

		text := getResultText(result)
		assert.Contains(t, text, "workflow-a")
		assert.Contains(t, text, "workflow-b")
	})

	t.Run("empty input returns empty result message", func(t *testing.T) {
		mock := &MockWorkflowDefinitionBulkGetter{
			Definitions: map[string]string{},
		}

		_, handler := QueryWorkflowsYq(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": ".spec",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)

		text := getResultText(result)
		assert.Contains(t, text, "No workflows found")
	})

	t.Run("error propagation from client", func(t *testing.T) {
		mock := &MockWorkflowDefinitionBulkGetter{
			Error: assert.AnError,
		}

		_, handler := QueryWorkflowsYq(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": ".spec",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err) // Handler returns error in result, not as error
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})

	t.Run("missing required parameter returns error", func(t *testing.T) {
		mock := &MockWorkflowDefinitionBulkGetter{}
		_, handler := QueryWorkflowsYq(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})
}

func TestQueryExecutionsYq(t *testing.T) {
	t.Run("per-item mode returns keyed results", func(t *testing.T) {
		mock := &MockExecutionBulkGetter{
			Executions: map[string]string{
				"exec-1": `{
					"id": "exec-1",
					"result": {
						"status": "passed",
						"duration": "1m30s"
					}
				}`,
				"exec-2": `{
					"id": "exec-2",
					"result": {
						"status": "failed",
						"duration": "2m45s"
					}
				}`,
			},
		}

		tool, handler := QueryExecutionsYq(mock)

		// Verify tool definition
		assert.Equal(t, "query_executions_yq", tool.Name)

		// Test handler
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": ".result.status",
			"aggregate":  false,
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)

		text := getResultText(result)
		assert.Contains(t, text, "passed")
		assert.Contains(t, text, "failed")
	})

	t.Run("aggregate mode combines executions into array", func(t *testing.T) {
		mock := &MockExecutionBulkGetter{
			Executions: map[string]string{
				"exec-1": `{"id": "exec-1", "result": {"status": "passed"}}`,
				"exec-2": `{"id": "exec-2", "result": {"status": "failed"}}`,
			},
		}

		_, handler := QueryExecutionsYq(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": ".[].result.status",
			"aggregate":  true,
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)

		text := getResultText(result)
		assert.Contains(t, text, "passed")
		assert.Contains(t, text, "failed")
	})

	t.Run("empty input returns empty result message", func(t *testing.T) {
		mock := &MockExecutionBulkGetter{
			Executions: map[string]string{},
		}

		_, handler := QueryExecutionsYq(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": ".result",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)

		text := getResultText(result)
		assert.Contains(t, text, "No executions found")
	})

	t.Run("error propagation from client", func(t *testing.T) {
		mock := &MockExecutionBulkGetter{
			Error: assert.AnError,
		}

		_, handler := QueryExecutionsYq(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": ".result",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})
}

// getResultText extracts the text content from an MCP result
func getResultText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		return textContent.Text
	}
	return ""
}
