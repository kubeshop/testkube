package jsonpath

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuery_BasicPaths(t *testing.T) {
	data := map[string]any{
		"name": "my-workflow",
		"spec": map[string]any{
			"steps": []any{
				map[string]any{"name": "step1"},
				map[string]any{"name": "step2"},
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "root element",
			path:     "$",
			expected: []any{data},
		},
		{
			name:     "direct property",
			path:     "$.name",
			expected: []any{"my-workflow"},
		},
		{
			name:     "nested property",
			path:     "$.spec.steps",
			expected: []any{data["spec"].(map[string]any)["steps"]},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Query(tt.path, data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuery_ArrayAccess(t *testing.T) {
	data := map[string]any{
		"items": []any{"a", "b", "c"},
		"nested": map[string]any{
			"list": []any{
				map[string]any{"id": 1},
				map[string]any{"id": 2},
				map[string]any{"id": 3},
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "first element",
			path:     "$.items[0]",
			expected: []any{"a"},
		},
		{
			name:     "last element by index",
			path:     "$.items[2]",
			expected: []any{"c"},
		},
		{
			name:     "all elements with wildcard",
			path:     "$.items[*]",
			expected: []any{"a", "b", "c"},
		},
		{
			name:     "nested array element property",
			path:     "$.nested.list[1].id",
			expected: []any{int64(2)},
		},
		{
			name:     "all ids from nested array",
			path:     "$.nested.list[*].id",
			expected: []any{int64(1), int64(2), int64(3)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Query(tt.path, data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuery_RecursiveDescent(t *testing.T) {
	data := map[string]any{
		"spec": map[string]any{
			"container": map[string]any{
				"image": "base:1.0",
			},
			"steps": []any{
				map[string]any{
					"name": "build",
					"run": map[string]any{
						"image": "python:3.12",
					},
				},
				map[string]any{
					"name": "test",
					"run": map[string]any{
						"image": "node:20",
					},
				},
			},
			"services": map[string]any{
				"db": map[string]any{
					"image": "postgres:16",
				},
				"cache": map[string]any{
					"image": "redis:7",
				},
			},
		},
	}

	tests := []struct {
		name        string
		path        string
		expectedLen int
		contains    []any
	}{
		{
			name:        "find all images",
			path:        "$..image",
			expectedLen: 5,
			contains:    []any{"base:1.0", "python:3.12", "node:20", "postgres:16", "redis:7"},
		},
		{
			name:        "find all names",
			path:        "$..name",
			expectedLen: 2,
			contains:    []any{"build", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Query(tt.path, data)
			require.NoError(t, err)
			assert.Len(t, result, tt.expectedLen)
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestQuery_Filters(t *testing.T) {
	data := map[string]any{
		"steps": []any{
			map[string]any{"name": "setup", "status": "passed"},
			map[string]any{"name": "test", "status": "failed"},
			map[string]any{"name": "cleanup", "status": "passed"},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name: "filter by equality",
			path: "$.steps[?(@.status == 'failed')]",
			expected: []any{
				map[string]any{"name": "test", "status": "failed"},
			},
		},
		{
			name: "filter by inequality",
			path: "$.steps[?(@.status != 'passed')]",
			expected: []any{
				map[string]any{"name": "test", "status": "failed"},
			},
		},
		{
			name:     "filter with no matches",
			path:     "$.steps[?(@.status == 'running')]",
			expected: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Query(tt.path, data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuery_MissingPaths(t *testing.T) {
	data := map[string]any{
		"name": "workflow",
		"spec": map[string]any{
			"steps": []any{},
		},
	}

	tests := []struct {
		name string
		path string
	}{
		{
			name: "missing property",
			path: "$.nonexistent",
		},
		{
			name: "missing nested property",
			path: "$.spec.services",
		},
		{
			name: "missing array index",
			path: "$.spec.steps[0]",
		},
		{
			name: "missing recursive property",
			path: "$..image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Query(tt.path, data)
			require.NoError(t, err, "missing paths should not return errors")
			assert.Empty(t, result, "missing paths should return empty slice")
			assert.NotNil(t, result, "result should be empty slice, not nil")
		})
	}
}

func TestQuery_InvalidSyntax(t *testing.T) {
	data := map[string]any{"name": "test"}

	tests := []struct {
		name string
		path string
	}{
		{
			name: "unclosed bracket",
			path: "$.items[0",
		},
		{
			name: "invalid filter syntax",
			path: "$.items[?(@.x ===)]",
		},
		{
			name: "invalid characters",
			path: "$.items[###]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Query(tt.path, data)
			require.Error(t, err, "invalid syntax should return error")
			assert.Nil(t, result)

			// Check error type
			var qErr *QueryError
			assert.ErrorAs(t, err, &qErr)
			assert.Equal(t, tt.path, qErr.Path)
			assert.Contains(t, qErr.Message, "invalid JSONPath syntax")
		})
	}
}

func TestQuery_NullHandling(t *testing.T) {
	tests := []struct {
		name string
		data any
		path string
	}{
		{
			name: "nil data",
			data: nil,
			path: "$.name",
		},
		{
			name: "null value in data",
			data: map[string]any{
				"name":  "test",
				"value": nil,
			},
			path: "$.value",
		},
		{
			name: "nested null",
			data: map[string]any{
				"outer": map[string]any{
					"inner": nil,
				},
			},
			path: "$.outer.inner.deep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Query(tt.path, tt.data)
			require.NoError(t, err, "null values should not cause errors")
			// Result may be empty or contain nil, but should not error
			assert.NotNil(t, result)
		})
	}
}

func TestQuery_WithContext(t *testing.T) {
	data := map[string]any{"name": "test"}

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := QueryWithContext(ctx, "$.name", data, DefaultOptions())
		require.Error(t, err)

		var qErr *QueryError
		require.ErrorAs(t, err, &qErr)
		assert.Contains(t, qErr.Message, "timed out")
	})

	t.Run("context with deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := QueryWithContext(ctx, "$.name", data, DefaultOptions())
		require.NoError(t, err)
		assert.Equal(t, []any{"test"}, result)
	})
}

func TestQuery_BracketNotation(t *testing.T) {
	data := map[string]any{
		"special-key": "value1",
		"another.key": "value2",
		"normal":      "value3",
	}

	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "bracket notation with hyphen",
			path:     "$['special-key']",
			expected: []any{"value1"},
		},
		{
			name:     "bracket notation with dot",
			path:     "$['another.key']",
			expected: []any{"value2"},
		},
		{
			name:     "bracket notation for normal key",
			path:     "$['normal']",
			expected: []any{"value3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Query(tt.path, data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuery_ComplexWorkflowExample(t *testing.T) {
	// Real-world workflow-like structure
	workflow := map[string]any{
		"apiVersion": "testworkflows.testkube.io/v1",
		"kind":       "TestWorkflow",
		"metadata": map[string]any{
			"name": "api-tests",
			"labels": map[string]any{
				"env":  "prod",
				"tool": "cypress",
			},
		},
		"spec": map[string]any{
			"content": map[string]any{
				"git": map[string]any{
					"uri":      "https://github.com/org/repo",
					"revision": "main",
				},
			},
			"steps": []any{
				map[string]any{
					"name": "setup",
					"run": map[string]any{
						"image": "node:20",
						"shell": "npm install",
					},
				},
				map[string]any{
					"name": "test",
					"run": map[string]any{
						"image": "cypress/included:13",
						"shell": "cypress run",
					},
				},
			},
			"services": map[string]any{
				"database": map[string]any{
					"image": "postgres:16",
					"env": []any{
						map[string]any{"name": "POSTGRES_DB", "value": "test"},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		path        string
		expected    []any
		expectedLen int
	}{
		{
			name:     "get workflow name",
			path:     "$.metadata.name",
			expected: []any{"api-tests"},
		},
		{
			name:     "get all labels",
			path:     "$.metadata.labels",
			expected: []any{map[string]any{"env": "prod", "tool": "cypress"}},
		},
		{
			name:     "get git uri",
			path:     "$.spec.content.git.uri",
			expected: []any{"https://github.com/org/repo"},
		},
		{
			name:     "get all step names",
			path:     "$.spec.steps[*].name",
			expected: []any{"setup", "test"},
		},
		{
			name:     "get step images",
			path:     "$.spec.steps[*].run.image",
			expected: []any{"node:20", "cypress/included:13"},
		},
		{
			name:        "find all images recursively",
			path:        "$..image",
			expectedLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Query(tt.path, workflow)
			require.NoError(t, err)
			if tt.expected != nil {
				assert.Equal(t, tt.expected, result)
			}
			if tt.expectedLen > 0 {
				assert.Len(t, result, tt.expectedLen)
			}
		})
	}
}

func TestQuery_ComplexExecutionExample(t *testing.T) {
	// Real-world execution-like structure
	execution := map[string]any{
		"id":          "67d2cdbc351aecb2720afdf2",
		"name":        "api-tests-42",
		"scheduledAt": "2025-01-15T10:30:00Z",
		"result": map[string]any{
			"status":   "failed",
			"duration": "5m30s",
			"steps": []any{
				map[string]any{
					"name":     "setup",
					"status":   "passed",
					"duration": "30s",
				},
				map[string]any{
					"name":         "test",
					"status":       "failed",
					"duration":     "4m50s",
					"errorMessage": "Test assertion failed",
				},
				map[string]any{
					"name":     "cleanup",
					"status":   "passed",
					"duration": "10s",
				},
			},
			"errorMessage": "Step 'test' failed",
		},
		"workflow": map[string]any{
			"name": "api-tests",
		},
	}

	tests := []struct {
		name     string
		path     string
		expected []any
	}{
		{
			name:     "get execution status",
			path:     "$.result.status",
			expected: []any{"failed"},
		},
		{
			name:     "get total duration",
			path:     "$.result.duration",
			expected: []any{"5m30s"},
		},
		{
			name:     "get all step names",
			path:     "$.result.steps[*].name",
			expected: []any{"setup", "test", "cleanup"},
		},
		{
			name:     "get all step statuses",
			path:     "$.result.steps[*].status",
			expected: []any{"passed", "failed", "passed"},
		},
		{
			name:     "get failed steps",
			path:     "$.result.steps[?(@.status == 'failed')].name",
			expected: []any{"test"},
		},
		{
			name:     "get workflow name",
			path:     "$.workflow.name",
			expected: []any{"api-tests"},
		},
		{
			name:     "get top-level error message",
			path:     "$.result.errorMessage",
			expected: []any{"Step 'test' failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Query(tt.path, execution)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Equal(t, DefaultTimeout, opts.Timeout)
	assert.Equal(t, DefaultMaxOutputSize, opts.MaxOutputSize)
	assert.Equal(t, DefaultMaxInputSize, opts.MaxInputSize)
}

func TestQueryError(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		err := &QueryError{
			Path:    "$.test",
			Message: "something went wrong",
			Err:     assert.AnError,
		}

		assert.Contains(t, err.Error(), "$.test")
		assert.Contains(t, err.Error(), "something went wrong")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("without underlying error", func(t *testing.T) {
		err := &QueryError{
			Path:    "$.test",
			Message: "something went wrong",
		}

		assert.Contains(t, err.Error(), "$.test")
		assert.Contains(t, err.Error(), "something went wrong")
		assert.Nil(t, err.Unwrap())
	})
}
