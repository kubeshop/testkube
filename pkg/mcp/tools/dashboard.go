package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

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
		mcp.WithString("stepRef",
			mcp.Description("Step reference ID to deep link to a specific workflow step in the log view (e.g. 'rwhc2zn'). Obtain from execution info signatures. Requires executionId."),
		),
		mcp.WithString("executionTab",
			mcp.Description("Execution tab to navigate to (e.g. 'log-output'). Defaults to 'log-output' when stepRef is provided."),
		),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resourceType := request.GetString("resourceType", "")
		workflowName := request.GetString("workflowName", "")
		executionID := request.GetString("executionId", "")
		stepRef := request.GetString("stepRef", "")
		executionTab := request.GetString("executionTab", "")

		if dashboardUrl == "" {
			return mcp.NewToolResultError("Dashboard URL is required"), nil
		}

		if workflowName == "" {
			return mcp.NewToolResultError("workflowName is required"), nil
		}

		if stepRef != "" && executionID == "" {
			return mcp.NewToolResultError("stepRef requires executionId to be set"), nil
		}

		// Default executionTab to log-output when stepRef is provided
		if stepRef != "" && executionTab == "" {
			executionTab = "log-output"
		}

		baseDashboardPath := fmt.Sprintf("/organization/%s/environment/%s/dashboard", orgId, envId)

		// Build URL based on resource type
		var resultURL string
		switch resourceType {
		case "workflow":
			workflowPath := fmt.Sprintf("%s/test-workflows/%s", baseDashboardPath, workflowName)
			// If executionId is provided, link directly to that execution
			if executionID != "" {
				workflowPath += fmt.Sprintf("/execution/%s", executionID)
				if executionTab != "" {
					workflowPath += fmt.Sprintf("/%s", executionTab)
				}
				if stepRef != "" {
					workflowPath += fmt.Sprintf("?ref=%s", url.QueryEscape(stepRef))
				}
			}
			resultURL = fmt.Sprintf("%s%s", dashboardUrl, workflowPath)
		case "execution":
			if executionID == "" {
				return mcp.NewToolResultError("executionId is required for execution URLs"), nil
			}
			executionPath := fmt.Sprintf("%s/test-workflows/%s/execution/%s", baseDashboardPath, workflowName, executionID)
			if executionTab != "" {
				executionPath += fmt.Sprintf("/%s", executionTab)
			}
			if stepRef != "" {
				executionPath += fmt.Sprintf("?ref=%s", url.QueryEscape(stepRef))
			}
			resultURL = fmt.Sprintf("%s%s", dashboardUrl, executionPath)
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unsupported resource type: %s. Use 'workflow' or 'execution'", resourceType)), nil
		}

		result := map[string]string{
			"url": resultURL,
		}

		jsonResponse, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error marshaling response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonResponse)), nil
	}

	return tool, handler
}
