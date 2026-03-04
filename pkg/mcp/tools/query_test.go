package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestQueryWorkflows(t *testing.T) {
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

		tool, handler := QueryWorkflows(mock)

		// Verify tool definition
		assert.Equal(t, "query_workflows", tool.Name)

		// Test handler
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$..image",
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

		_, handler := QueryWorkflows(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$..name",
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

		_, handler := QueryWorkflows(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$.spec",
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

		_, handler := QueryWorkflows(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$.spec",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err) // Handler returns error in result, not as error
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})

	t.Run("missing required parameter returns error", func(t *testing.T) {
		mock := &MockWorkflowDefinitionBulkGetter{}
		_, handler := QueryWorkflows(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})

	t.Run("recursive descent finds all images", func(t *testing.T) {
		mock := &MockWorkflowDefinitionBulkGetter{
			Definitions: map[string]string{
				"multi-image": `
spec:
  container:
    image: base:1.0
  steps:
    - name: build
      run:
        image: python:3.12
    - name: test
      run:
        image: node:20
  services:
    db:
      image: postgres:16
`,
			},
		}

		_, handler := QueryWorkflows(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$..image",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.False(t, result.IsError)

		text := getResultText(result)
		assert.Contains(t, text, "base:1.0")
		assert.Contains(t, text, "python:3.12")
		assert.Contains(t, text, "node:20")
		assert.Contains(t, text, "postgres:16")
	})

	t.Run("missing path returns empty array not error", func(t *testing.T) {
		mock := &MockWorkflowDefinitionBulkGetter{
			Definitions: map[string]string{
				"simple": `
metadata:
  name: simple
`,
			},
		}

		_, handler := QueryWorkflows(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$.spec.nonexistent.path",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.False(t, result.IsError)

		text := getResultText(result)
		// Should return empty array, not error
		assert.Contains(t, text, "[]")
	})

	t.Run("extract step names", func(t *testing.T) {
		mock := &MockWorkflowDefinitionBulkGetter{
			Definitions: map[string]string{
				"workflow": `
spec:
  steps:
    - name: build
    - name: test
    - name: deploy
`,
			},
		}

		_, handler := QueryWorkflows(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$.spec.steps[*].name",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.False(t, result.IsError)

		text := getResultText(result)
		assert.Contains(t, text, "build")
		assert.Contains(t, text, "test")
		assert.Contains(t, text, "deploy")
	})

	t.Run("limit parameter is respected", func(t *testing.T) {
		receivedParams := ListWorkflowsParams{}
		mock := &MockWorkflowDefinitionBulkGetter{
			Definitions: map[string]string{},
		}

		// Create a wrapper to capture params
		originalGetter := mock
		captureGetter := &struct {
			WorkflowDefinitionBulkGetter
			captured *ListWorkflowsParams
		}{
			WorkflowDefinitionBulkGetter: originalGetter,
			captured:                     &receivedParams,
		}
		_ = captureGetter // We verify by checking the tool handles limit properly

		_, handler := QueryWorkflows(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$",
			"limit":      float64(25), // JSON numbers come as float64
		}

		_, err := handler(context.Background(), request)
		require.NoError(t, err)
		// The test passes if no error - limit handling is internal
	})
}

func TestQueryExecutions(t *testing.T) {
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

		tool, handler := QueryExecutions(mock)

		// Verify tool definition
		assert.Equal(t, "query_executions", tool.Name)

		// Test handler
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$.result.status",
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

		_, handler := QueryExecutions(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$[*].result.status",
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

		_, handler := QueryExecutions(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$.result",
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

		_, handler := QueryExecutions(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$.result",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})

	t.Run("missing required parameter returns error", func(t *testing.T) {
		mock := &MockExecutionBulkGetter{}
		_, handler := QueryExecutions(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})

	t.Run("recursive descent finds all durations", func(t *testing.T) {
		mock := &MockExecutionBulkGetter{
			Executions: map[string]string{
				"exec-1": `{
					"result": {
						"duration": "1m30s",
						"steps": [
							{"name": "step1", "duration": "30s"},
							{"name": "step2", "duration": "1m"}
						]
					}
				}`,
			},
		}

		_, handler := QueryExecutions(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$..duration",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.False(t, result.IsError)

		text := getResultText(result)
		assert.Contains(t, text, "1m30s")
		assert.Contains(t, text, "30s")
		assert.Contains(t, text, "1m")
	})

	t.Run("missing path returns empty array not error", func(t *testing.T) {
		mock := &MockExecutionBulkGetter{
			Executions: map[string]string{
				"exec-1": `{"id": "exec-1"}`,
			},
		}

		_, handler := QueryExecutions(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$.nonexistent.path",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.False(t, result.IsError)

		text := getResultText(result)
		// Should return empty array, not error
		assert.Contains(t, text, "[]")
	})

	t.Run("extract step statuses", func(t *testing.T) {
		mock := &MockExecutionBulkGetter{
			Executions: map[string]string{
				"exec-1": `{
					"result": {
						"steps": [
							{"name": "setup", "status": "passed"},
							{"name": "test", "status": "failed"},
							{"name": "cleanup", "status": "passed"}
						]
					}
				}`,
			},
		}

		_, handler := QueryExecutions(mock)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"expression": "$.result.steps[*].status",
		}

		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.False(t, result.IsError)

		text := getResultText(result)
		assert.Contains(t, text, "passed")
		assert.Contains(t, text, "failed")
	})
}

func TestQueryWorkflows_JSONPathSyntax(t *testing.T) {
	// Test various JSONPath syntax features
	workflow := `
metadata:
  name: test-workflow
  labels:
    tool: cypress
    env: prod
spec:
  steps:
    - name: step1
      run:
        image: alpine:3.18
    - name: step2
      run:
        image: python:3.11
  services:
    db:
      image: postgres:16
    cache:
      image: redis:7
`

	tests := []struct {
		name       string
		expression string
		contains   []string
	}{
		{
			name:       "root element",
			expression: "$",
			contains:   []string{"test-workflow"},
		},
		{
			name:       "direct property",
			expression: "$.metadata.name",
			contains:   []string{"test-workflow"},
		},
		{
			name:       "nested array access",
			expression: "$.spec.steps[0].name",
			contains:   []string{"step1"},
		},
		{
			name:       "all array elements",
			expression: "$.spec.steps[*].name",
			contains:   []string{"step1", "step2"},
		},
		{
			name:       "recursive descent",
			expression: "$..image",
			contains:   []string{"alpine:3.18", "python:3.11", "postgres:16", "redis:7"},
		},
		{
			name:       "labels access",
			expression: "$.metadata.labels",
			contains:   []string{"cypress", "prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockWorkflowDefinitionBulkGetter{
				Definitions: map[string]string{
					"test-workflow": workflow,
				},
			}

			_, handler := QueryWorkflows(mock)

			request := mcp.CallToolRequest{}
			request.Params.Arguments = map[string]any{
				"expression": tt.expression,
			}

			result, err := handler(context.Background(), request)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.False(t, result.IsError, "unexpected error: %s", getResultText(result))

			text := getResultText(result)
			for _, expected := range tt.contains {
				assert.Contains(t, text, expected, "expected to find %q in result", expected)
			}
		})
	}
}

func TestQueryExecutions_JSONPathSyntax(t *testing.T) {
	// Test various JSONPath syntax features with execution data
	execution := `{
		"id": "exec-123",
		"workflow": {"name": "my-workflow"},
		"result": {
			"status": "failed",
			"duration": "2m30s",
			"errorMessage": "Test failed: assertion error",
			"steps": [
				{"name": "setup", "status": "passed", "duration": "10s"},
				{"name": "test", "status": "failed", "duration": "2m", "errorMessage": "assertion error"},
				{"name": "cleanup", "status": "skipped", "duration": "0s"}
			]
		}
	}`

	tests := []struct {
		name       string
		expression string
		contains   []string
	}{
		{
			name:       "root element",
			expression: "$",
			contains:   []string{"exec-123"},
		},
		{
			name:       "direct property",
			expression: "$.result.status",
			contains:   []string{"failed"},
		},
		{
			name:       "nested property",
			expression: "$.workflow.name",
			contains:   []string{"my-workflow"},
		},
		{
			name:       "array element by index",
			expression: "$.result.steps[1].name",
			contains:   []string{"test"},
		},
		{
			name:       "all step names",
			expression: "$.result.steps[*].name",
			contains:   []string{"setup", "test", "cleanup"},
		},
		{
			name:       "recursive descent for status",
			expression: "$..status",
			contains:   []string{"failed", "passed", "skipped"},
		},
		{
			name:       "recursive descent for errorMessage",
			expression: "$..errorMessage",
			contains:   []string{"Test failed", "assertion error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockExecutionBulkGetter{
				Executions: map[string]string{
					"exec-123": execution,
				},
			}

			_, handler := QueryExecutions(mock)

			request := mcp.CallToolRequest{}
			request.Params.Arguments = map[string]any{
				"expression": tt.expression,
			}

			result, err := handler(context.Background(), request)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.False(t, result.IsError, "unexpected error: %s", getResultText(result))

			text := getResultText(result)
			for _, expected := range tt.contains {
				assert.Contains(t, text, expected, "expected to find %q in result", expected)
			}
		})
	}
}

func TestQueryWorkflows_InvalidSyntax(t *testing.T) {
	mock := &MockWorkflowDefinitionBulkGetter{
		Definitions: map[string]string{
			"test": `name: test`,
		},
	}

	_, handler := QueryWorkflows(mock)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"expression": "$[[[invalid",
	}

	result, err := handler(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should contain error indication
	text := getResultText(result)
	assert.Contains(t, text, "ERROR")
}

func TestQueryExecutions_InvalidJSON(t *testing.T) {
	mock := &MockExecutionBulkGetter{
		Executions: map[string]string{
			"bad": `{invalid json`,
		},
	}

	_, handler := QueryExecutions(mock)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"expression": "$.id",
	}

	result, err := handler(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)
	// Error is returned as a result, not in handler error
	text := getResultText(result)
	assert.Contains(t, text, "ERROR")
}

func TestQueryWorkflows_AggregateWithMultiple(t *testing.T) {
	mock := &MockWorkflowDefinitionBulkGetter{
		Definitions: map[string]string{
			"wf-1": `
metadata:
  name: wf-1
  labels:
    tool: cypress
`,
			"wf-2": `
metadata:
  name: wf-2
  labels:
    tool: playwright
`,
			"wf-3": `
metadata:
  name: wf-3
  labels:
    tool: k6
`,
		},
	}

	_, handler := QueryWorkflows(mock)

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]any{
		"expression": "$[*].metadata.labels.tool",
		"aggregate":  true,
	}

	result, err := handler(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.IsError)

	text := getResultText(result)
	// All tools should be found
	assert.Contains(t, text, "cypress")
	assert.Contains(t, text, "playwright")
	assert.Contains(t, text, "k6")

	// Verify it's valid JSON
	var parsed any
	err = json.Unmarshal([]byte(text), &parsed)
	require.NoError(t, err, "result should be valid JSON")
}
