package tools

import (
	"context"
	_ "embed"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

//go:embed schemas/workflow_schema.yaml
var workflowSchema string

//go:embed schemas/template_schema.yaml
var templateSchema string

//go:embed schemas/execution_schema.yaml
var executionSchema string

const (
	GetWorkflowSchemaDescription = `Get the YAML schema for TestWorkflow definitions. Returns a structured schema 
showing all available fields, their types, and descriptions. Use this schema to understand which JSONPath 
expressions will work when querying workflows with query_workflows.

Common paths based on this schema:
  $.metadata.name              - Workflow name
  $.metadata.labels            - Workflow labels
  $.spec.steps[*].name         - All step names
  $.spec.steps[*].shell        - Shell commands in steps
  $.spec.steps[*].run.image    - Container images in run steps
  $.spec.container.image       - Default container image
  $.spec.content.git.uri       - Git repository URL
  $.spec.services              - Service definitions
  $..image                     - All image fields (recursive)

The schema uses YAML comments to describe each field's type and purpose.`

	GetTemplateSchemaDescription = `Get the YAML schema for TestWorkflowTemplate definitions. Returns a structured 
schema showing all available fields, their types, and descriptions. Templates are reusable workflow components 
that can be included in multiple workflows.

Common paths based on this schema:
  $.metadata.name              - Template name
  $.spec.steps[*].name         - All step names
  $.spec.container.image       - Default container image
  $.spec.config                - Template configuration parameters

The schema uses YAML comments to describe each field's type and purpose.`

	GetExecutionSchemaDescription = `Get the YAML schema for TestWorkflowExecution data. Returns a structured schema 
showing all available fields, their types, and descriptions. Use this schema to understand which JSONPath 
expressions will work when querying executions with query_executions.

Common paths based on this schema:
  $.id                         - Execution ID
  $.name                       - Execution name
  $.result.status              - Execution status (passed, failed, running, etc.)
  $.result.duration            - Human-readable duration
  $.result.durationMs          - Duration in milliseconds
  $.result.steps               - Map of step results (keyed by step ref)
  $.result.startedAt           - When execution started
  $.result.finishedAt          - When execution finished
  $.workflow.metadata.name     - Workflow name
  $.tags                       - Execution tags
  $.configParams               - Configuration parameters passed to execution

The schema uses YAML comments to describe each field's type and purpose.`
)

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

// GetTemplateSchema returns a tool that provides the TestWorkflowTemplate YAML schema.
// This tool has no parameters and returns static embedded content.
func GetTemplateSchema() (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_template_schema",
		mcp.WithDescription(GetTemplateSchemaDescription),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(templateSchema), nil
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
