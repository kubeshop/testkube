package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func BuildDashboardUrl(dashboardUrl string, orgId string, envId string) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("build_dashboard_url",
		mcp.WithDescription(BuildDashboardUrlDescription),
		mcp.WithString("resourceType",
			mcp.Required(),
			mcp.Description("Type of dashboard resource: 'workflow' or 'execution'"),
		),
		mcp.WithString("workflowName",
			mcp.Required(),
			mcp.Description("Name of the test workflow"),
		),
		mcp.WithString("executionId",
			mcp.Description("Execution ID (required for execution URLs)"),
		),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resourceType := request.GetString("resourceType", "")
		workflowName := request.GetString("workflowName", "")
		executionID := request.GetString("executionId", "")

		if dashboardUrl == "" {
			return mcp.NewToolResultError("Dashboard URL is required"), nil
		}

		if workflowName == "" {
			return mcp.NewToolResultError("workflowName is required"), nil
		}

		baseDashboardPath := fmt.Sprintf("/organization/%s/environment/%s/dashboard", orgId, envId)

		// Build URL based on resource type
		var url string
		switch resourceType {
		case "workflow":
			workflowPath := fmt.Sprintf("%s/test-workflows/%s", baseDashboardPath, workflowName)
			// If executionId is provided, link directly to that execution
			if executionID != "" {
				workflowPath += fmt.Sprintf("/execution/%s", executionID)
			}
			url = fmt.Sprintf("%s%s", dashboardUrl, workflowPath)
		case "execution":
			if executionID == "" {
				return mcp.NewToolResultError("executionId is required for execution URLs"), nil
			}
			executionPath := fmt.Sprintf("%s/test-workflows/%s/execution/%s", baseDashboardPath, workflowName, executionID)
			url = fmt.Sprintf("%s%s", dashboardUrl, executionPath)
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unsupported resource type: %s. Use 'workflow' or 'execution'", resourceType)), nil
		}

		result := map[string]string{
			"url": url,
		}

		jsonResponse, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error marshaling response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonResponse)), nil
	}

	return tool, handler
}
