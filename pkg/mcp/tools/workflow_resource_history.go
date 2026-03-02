package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kubeshop/testkube/pkg/mcp/formatters"
)

// WorkflowResourceHistoryParams holds parameters for the GetWorkflowResourceHistory tool.
type WorkflowResourceHistoryParams struct {
	WorkflowName string
	LastN        int
	Metrics      string // comma-separated: cpu,memory,disk,network or empty for all
}

// WorkflowResourceHistoryGetter is the interface for fetching execution resource history.
type WorkflowResourceHistoryGetter interface {
	GetWorkflowResourceHistory(ctx context.Context, params WorkflowResourceHistoryParams) (string, error)
}

// GetWorkflowResourceHistory creates an MCP tool that aggregates resource metrics
// across multiple executions of a workflow.
func GetWorkflowResourceHistory(client WorkflowResourceHistoryGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_workflow_resource_history",
		mcp.WithDescription(GetWorkflowResourceHistoryDescription),
		mcp.WithString("workflowName", mcp.Required(), mcp.Description(WorkflowNameDescription)),
		mcp.WithString("lastN", mcp.Description("Number of recent executions to analyze (default: 50, max: 100)")),
		mcp.WithString("metrics", mcp.Description("Filter to specific metrics: cpu, memory, disk, network. Comma-separated for multiple. Default: all metrics.")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflowName, err := RequiredParam[string](request, "workflowName")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		params := WorkflowResourceHistoryParams{
			WorkflowName: workflowName,
			LastN:        50, // default
		}

		if lastNStr := request.GetString("lastN", ""); lastNStr != "" {
			if lastN, err := strconv.Atoi(lastNStr); err == nil && lastN > 0 {
				if lastN > 100 {
					lastN = 100
				}
				params.LastN = lastN
			}
		}

		params.Metrics = request.GetString("metrics", "")

		result, err := client.GetWorkflowResourceHistory(ctx, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get workflow resource history: %v", err)), nil
		}

		formatted, err := formatters.FormatWorkflowResourceHistory(result, params.Metrics)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format resource history: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}
