package tools

import (
	"context"
	_ "embed"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

//go:embed schemas/workflow_schema.yaml
var workflowSchema string

//go:embed schemas/execution_schema.yaml
var executionSchema string

// GetWorkflowSchema returns a tool that provides the TestWorkflow YAML schema.
// This tool has no parameters and returns static embedded content.
func GetWorkflowSchema() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_workflow_schema",
		mcp.WithDescription(GetWorkflowSchemaDescription),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(workflowSchema), nil
	}

	return tool, handler
}

// GetExecutionSchema returns a tool that provides the TestWorkflowExecution YAML schema.
// This tool has no parameters and returns static embedded content.
func GetExecutionSchema() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_execution_schema",
		mcp.WithDescription(GetExecutionSchemaDescription),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(executionSchema), nil
	}

	return tool, handler
}
