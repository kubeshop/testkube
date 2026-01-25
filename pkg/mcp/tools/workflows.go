package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// convertConfigValuesToStrings converts all values in the config map to strings
// to avoid API errors when passing configuration parameters to workflows
func convertConfigValuesToStrings(config map[string]any) map[string]any {
	result := make(map[string]any)
	for key, value := range config {
		switch v := value.(type) {
		case string:
			result[key] = v
		case int:
			result[key] = strconv.Itoa(v)
		case int64:
			result[key] = strconv.FormatInt(v, 10)
		case float64:
			result[key] = strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			result[key] = strconv.FormatBool(v)
		default:
			// For any other type, convert to string using fmt.Sprintf
			result[key] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

type ListWorkflowsParams struct {
	ResourceGroup string
	Selector      string
	TextSearch    string
	PageSize      int
	Page          int
	Status        string
	GroupID       string
}

type WorkflowLister interface {
	ListWorkflows(ctx context.Context, params ListWorkflowsParams) (string, error)
}

func ListWorkflows(client WorkflowLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_workflows",
		mcp.WithDescription(ListWorkflowsDescription),
		mcp.WithString("resourceGroup", mcp.Description(ResourceGroupDescription)),
		mcp.WithString("selector", mcp.Description(SelectorDescription)),
		mcp.WithString("textSearch", mcp.Description(TextSearchDescription)),
		mcp.WithString("pageSize", mcp.Description(PageSizeDescription)),
		mcp.WithString("page", mcp.Description(PageDescription)),
		mcp.WithString("status", mcp.Description(StatusDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := ListWorkflowsParams{
			ResourceGroup: request.GetString("resourceGroup", ""),
			Selector:      request.GetString("selector", ""),
			TextSearch:    request.GetString("textSearch", ""),
			Status:        request.GetString("status", ""),
		}

		if pageSizeStr := request.GetString("pageSize", "10"); pageSizeStr != "" {
			if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
				params.PageSize = pageSize
			}
		}
		if pageStr := request.GetString("page", "0"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil && page >= 0 {
				params.Page = page
			}
		}

		result, err := client.ListWorkflows(ctx, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list workflows: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

type WorkflowCreator interface {
	CreateWorkflow(ctx context.Context, workflowDefinition string) (string, error)
}

func CreateWorkflow(client WorkflowCreator) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("create_workflow",
		mcp.WithDescription(CreateWorkflowDescription),
		mcp.WithString("yaml", mcp.Required(), mcp.Description("Complete YAML definition of the TestWorkflow to create in Testkube. This should be the full workflow specification including metadata, spec, and all steps.")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		yaml, err := RequiredParam[string](request, "yaml")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := client.CreateWorkflow(ctx, yaml)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create workflow: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

type WorkflowDefinitionGetter interface {
	GetWorkflowDefinition(ctx context.Context, workflowName string) (string, error)
}

func GetWorkflowDefinition(client WorkflowDefinitionGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_workflow_definition",
		mcp.WithDescription(GetWorkflowDefinitionDescription),
		mcp.WithString("workflowName", mcp.Required(), mcp.Description(WorkflowNameDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflowName, err := RequiredParam[string](request, "workflowName")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := client.GetWorkflowDefinition(ctx, workflowName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get workflow definition: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

type WorkflowGetter interface {
	GetWorkflow(ctx context.Context, workflowName string) (string, error)
}

func GetWorkflow(client WorkflowGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_workflow",
		mcp.WithDescription(GetWorkflowDescription),
		mcp.WithString("workflowName", mcp.Required(), mcp.Description(WorkflowNameDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflowName, err := RequiredParam[string](request, "workflowName")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := client.GetWorkflow(ctx, workflowName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get workflow: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

type RunWorkflowParams struct {
	WorkflowName string
	Config       map[string]any
	Target       map[string]any
}

type WorkflowRunner interface {
	RunWorkflow(ctx context.Context, params RunWorkflowParams) (string, error)
}

func RunWorkflow(client WorkflowRunner) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("run_workflow",
		mcp.WithDescription(RunWorkflowDescription),
		mcp.WithString("workflowName", mcp.Required(), mcp.Description(WorkflowNameDescription)),
		mcp.WithObject("config",
			mcp.Description("Configuration parameters for the workflow. Use get_workflow_definition tool first to examine the spec.config section to see what parameters are available.")),
		mcp.WithObject("target",
			mcp.Description("Target specification for multi-agent execution (optional). Supports: {\"name\": \"agent-name\"} for name-based targeting, {\"labels\": {\"env\": \"prod\"}} for label-based targeting, or standard ExecutionTarget format with match/not/replicate fields.")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflowName, err := RequiredParam[string](request, "workflowName")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		config := make(map[string]any)
		if configValue, ok, err := OptionalParamOK[map[string]any](request, "config"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		} else if ok {
			// Convert all config values to strings to avoid API errors
			config = convertConfigValuesToStrings(configValue)
		}

		var target map[string]any
		if targetValue, ok, err := OptionalParamOK[map[string]any](request, "target"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		} else if ok {
			target = targetValue
		}

		params := RunWorkflowParams{
			WorkflowName: workflowName,
			Config:       config,
			Target:       target,
		}

		result, err := client.RunWorkflow(ctx, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to run workflow: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

type WorkflowUpdater interface {
	UpdateWorkflow(ctx context.Context, workflowName, workflowDefinition string) (string, error)
}

func UpdateWorkflow(client WorkflowUpdater) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("update_workflow",
		mcp.WithDescription(UpdateWorkflowDescription),
		mcp.WithString("workflowName", mcp.Required(), mcp.Description(WorkflowNameDescription)),
		mcp.WithString("yaml", mcp.Required(), mcp.Description("Complete YAML definition of the TestWorkflow to update in Testkube. This should be the full workflow specification including metadata, spec, and all steps.")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflowName, err := RequiredParam[string](request, "workflowName")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		yaml, err := RequiredParam[string](request, "yaml")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := client.UpdateWorkflow(ctx, workflowName, yaml)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update workflow: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

type WorkflowMetricsGetter interface {
	GetWorkflowMetrics(ctx context.Context, workflowName string) (string, error)
}

func GetWorkflowMetrics(client WorkflowMetricsGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_workflow_metrics",
		mcp.WithDescription(GetWorkflowMetricsDescription),
		mcp.WithString("workflowName", mcp.Required(), mcp.Description(WorkflowNameDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflowName := request.GetString("workflowName", "")
		if workflowName == "" {
			return mcp.NewToolResultError("workflowName parameter is required"), nil
		}

		result, err := client.GetWorkflowMetrics(ctx, workflowName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get workflow metrics: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

type WorkflowExecutionMetricsGetter interface {
	GetWorkflowExecutionMetrics(ctx context.Context, workflowName, executionID string) (string, error)
}

func GetWorkflowExecutionMetrics(client WorkflowExecutionMetricsGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_workflow_execution_metrics",
		mcp.WithDescription(GetWorkflowExecutionMetricsDescription),
		mcp.WithString("workflowName", mcp.Required(), mcp.Description(WorkflowNameDescription)),
		mcp.WithString("executionId", mcp.Required(), mcp.Description(ExecutionIdDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflowName, err := RequiredParam[string](request, "workflowName")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		executionID, err := RequiredParam[string](request, "executionId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := client.GetWorkflowExecutionMetrics(ctx, workflowName, executionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get workflow execution metrics: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}
