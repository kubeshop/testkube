package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWorkflowSchema(t *testing.T) {
	tool, handler := GetWorkflowSchema()

	t.Run("tool has correct name and description", func(t *testing.T) {
		assert.Equal(t, "get_workflow_schema", tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.Contains(t, tool.Description, "TestWorkflow")
	})

	t.Run("returns workflow schema content", func(t *testing.T) {
		request := mcp.CallToolRequest{}
		result, err := handler(context.Background(), request)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok, "expected TextContent")

		// Verify essential workflow schema elements are present
		assert.Contains(t, textContent.Text, "apiVersion")
		assert.Contains(t, textContent.Text, "TestWorkflow")
		assert.Contains(t, textContent.Text, "metadata")
		assert.Contains(t, textContent.Text, "spec")
		assert.Contains(t, textContent.Text, "steps")
		assert.Contains(t, textContent.Text, "container")
		assert.Contains(t, textContent.Text, "image")
	})

	t.Run("schema contains key workflow fields", func(t *testing.T) {
		request := mcp.CallToolRequest{}
		result, err := handler(context.Background(), request)

		require.NoError(t, err)
		textContent := result.Content[0].(mcp.TextContent)

		// Check for important workflow fields
		keyFields := []string{
			"content",
			"git",
			"shell",
			"run",
			"services",
			"config",
			"after",
			"setup",
			"env",
			"artifacts",
		}

		for _, field := range keyFields {
			assert.True(t, strings.Contains(textContent.Text, field),
				"expected schema to contain field: %s", field)
		}
	})
}

func TestGetTemplateSchema(t *testing.T) {
	tool, handler := GetTemplateSchema()

	t.Run("tool has correct name and description", func(t *testing.T) {
		assert.Equal(t, "get_template_schema", tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.Contains(t, tool.Description, "TestWorkflowTemplate")
	})

	t.Run("returns template schema content", func(t *testing.T) {
		request := mcp.CallToolRequest{}
		result, err := handler(context.Background(), request)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok, "expected TextContent")

		// Verify essential template schema elements are present
		assert.Contains(t, textContent.Text, "apiVersion")
		assert.Contains(t, textContent.Text, "TestWorkflowTemplate")
		assert.Contains(t, textContent.Text, "metadata")
		assert.Contains(t, textContent.Text, "spec")
		assert.Contains(t, textContent.Text, "steps")
	})

	t.Run("schema contains key template fields", func(t *testing.T) {
		request := mcp.CallToolRequest{}
		result, err := handler(context.Background(), request)

		require.NoError(t, err)
		textContent := result.Content[0].(mcp.TextContent)

		// Check for important template fields
		keyFields := []string{
			"container",
			"config",
			"steps",
			"setup",
		}

		for _, field := range keyFields {
			assert.True(t, strings.Contains(textContent.Text, field),
				"expected schema to contain field: %s", field)
		}
	})
}

func TestGetExecutionSchema(t *testing.T) {
	tool, handler := GetExecutionSchema()

	t.Run("tool has correct name and description", func(t *testing.T) {
		assert.Equal(t, "get_execution_schema", tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.Contains(t, tool.Description, "TestWorkflowExecution")
	})

	t.Run("returns execution schema content", func(t *testing.T) {
		request := mcp.CallToolRequest{}
		result, err := handler(context.Background(), request)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok, "expected TextContent")

		// Verify essential execution schema elements are present
		assert.Contains(t, textContent.Text, "id")
		assert.Contains(t, textContent.Text, "name")
		assert.Contains(t, textContent.Text, "result")
		assert.Contains(t, textContent.Text, "status")
	})

	t.Run("schema contains key execution fields", func(t *testing.T) {
		request := mcp.CallToolRequest{}
		result, err := handler(context.Background(), request)

		require.NoError(t, err)
		textContent := result.Content[0].(mcp.TextContent)

		// Check for important execution fields
		keyFields := []string{
			"id",
			"name",
			"namespace",
			"scheduledAt",
			"result",
			"status",
			"duration",
			"durationMs",
			"steps",
			"workflow",
			"tags",
			"configParams",
			"signature",
			"output",
			"reports",
		}

		for _, field := range keyFields {
			assert.True(t, strings.Contains(textContent.Text, field),
				"expected schema to contain field: %s", field)
		}
	})

	t.Run("schema documents status values", func(t *testing.T) {
		request := mcp.CallToolRequest{}
		result, err := handler(context.Background(), request)

		require.NoError(t, err)
		textContent := result.Content[0].(mcp.TextContent)

		// Check that status values are documented
		statusValues := []string{
			"passed",
			"failed",
			"running",
			"queued",
			"aborted",
		}

		for _, status := range statusValues {
			assert.True(t, strings.Contains(textContent.Text, status),
				"expected schema to document status value: %s", status)
		}
	})
}

func TestSchemaToolsNoParameters(t *testing.T) {
	// All schema tools should work with empty requests (no parameters needed)
	tools := []struct {
		name    string
		getTool func() (mcp.Tool, server.ToolHandlerFunc)
	}{
		{"get_workflow_schema", GetWorkflowSchema},
		{"get_template_schema", GetTemplateSchema},
		{"get_execution_schema", GetExecutionSchema},
	}

	for _, tt := range tools {
		t.Run(tt.name+" requires no parameters", func(t *testing.T) {
			tool, handler := tt.getTool()

			// Tool should have no required inputs
			assert.Empty(t, tool.InputSchema.Required, "schema tools should have no required parameters")

			// Handler should work with empty request
			request := mcp.CallToolRequest{}
			result, err := handler(context.Background(), request)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Content)

			// Result should not be an error
			assert.False(t, result.IsError, "schema tool should not return error")
		})
	}
}

func TestSchemaEmbedding(t *testing.T) {
	// Verify that schemas are properly embedded and non-empty
	t.Run("workflow schema is embedded", func(t *testing.T) {
		assert.NotEmpty(t, workflowSchema, "workflow schema should be embedded")
		assert.True(t, len(workflowSchema) > 1000, "workflow schema should be substantial")
	})

	t.Run("template schema is embedded", func(t *testing.T) {
		assert.NotEmpty(t, templateSchema, "template schema should be embedded")
		assert.True(t, len(templateSchema) > 1000, "template schema should be substantial")
	})

	t.Run("execution schema is embedded", func(t *testing.T) {
		assert.NotEmpty(t, executionSchema, "execution schema should be embedded")
		assert.True(t, len(executionSchema) > 500, "execution schema should be substantial")
	})
}
