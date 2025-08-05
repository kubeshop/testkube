package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ExecutionLogger interface {
	GetExecutionLogs(ctx context.Context, executionId string) (string, error)
}

func FetchExecutionLogs(client ExecutionLogger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("fetch_execution_logs",
		mcp.WithDescription("Retrieves the full logs of a test workflow execution for debugging and analysis."),
		mcp.WithString("executionId",
			mcp.Required(),
			mcp.Description("The unique execution ID in MongoDB format (e.g., '67d2cdbc351aecb2720afdf2')."),
		),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionID := request.GetString("executionId", "")

		logs, err := client.GetExecutionLogs(ctx, executionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch logs: %v", err)), nil
		}

		return mcp.NewToolResultText(logs), nil
	}

	return tool, handler
}

type ListExecutionsParams struct {
	WorkflowName string
	Selector     string
	TextSearch   string
	PageSize     int
	Page         int
	Status       string
	Since        string
}

type ExecutionLister interface {
	ListExecutions(ctx context.Context, params ListExecutionsParams) (string, error)
}

func ListExecutions(client ExecutionLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_executions",
		mcp.WithDescription("List executions for a specific test workflow with filtering and pagination options. Returns execution summaries with status, timing, and results."),
		mcp.WithString("workflowName", mcp.Required(), mcp.Description(WorkflowNameDescription)),
		mcp.WithString("pageSize", mcp.Description(PageSizeDescription)),
		mcp.WithString("page", mcp.Description(PageDescription)),
		mcp.WithString("textSearch", mcp.Description(TextSearchDescription)),
		mcp.WithString("status", mcp.Description(StatusDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := ListExecutionsParams{
			WorkflowName: request.GetString("workflowName", ""),
			TextSearch:   request.GetString("textSearch", ""),
			Status:       request.GetString("status", ""),
		}

		if pageSizeStr := request.GetString("pageSize", "10"); pageSizeStr != "" {
			if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
				params.PageSize = pageSize
			}
		}
		if pageStr := request.GetString("page", "1"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
				params.Page = page
			}
		}

		result, err := client.ListExecutions(ctx, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list executions: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

type ExecutionInfoGetter interface {
	GetExecutionInfo(ctx context.Context, workflowName, executionId string) (string, error)
}

func GetExecutionInfo(client ExecutionInfoGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_execution_info",
		mcp.WithDescription("Get detailed information about a specific test workflow execution, including status, timing, results, and configuration."),
		mcp.WithString("workflowName", mcp.Required(), mcp.Description(WorkflowNameDescription)),
		mcp.WithString("executionId", mcp.Required(), mcp.Description(ExecutionIdDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflowName, err := RequiredParam[string](request, "workflowName")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		executionId, err := RequiredParam[string](request, "executionId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := client.GetExecutionInfo(ctx, workflowName, executionId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get execution info: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

type ExecutionLookup interface {
	LookupExecutionID(ctx context.Context, executionName string) (string, error)
}

func LookupExecutionId(client ExecutionLookup) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("lookup_execution_id",
		mcp.WithDescription("Resolves an execution name to its corresponding execution ID. Use this tool when you have an execution name (e.g., 'my-workflow-123', 'my-test-987-1') but need the execution ID. Many other tools require execution IDs (MongoDB format) rather than names."),
		mcp.WithString("executionName", mcp.Required(), mcp.Description(ExecutionNameDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionName, err := RequiredParam[string](request, "executionName")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if !isValidExecutionName(executionName) {
			return mcp.NewToolResultError(fmt.Sprintf("fnvalid execution name format: \"%s\" expected format: \"workflow-name-number\".", executionName)), nil
		}

		result, err := client.LookupExecutionID(ctx, executionName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to lookup execution ID: %v", err)), nil
		}

		executionID, err := extractExecutionIdFromResponse(result, executionName)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(executionID), nil
	}

	return tool, handler
}

func isValidExecutionName(executionName string) bool {
	lastDashIndex := strings.LastIndex(executionName, "-")
	if lastDashIndex == -1 {
		return false
	}

	executionNumberStr := executionName[lastDashIndex+1:]
	matched, _ := regexp.MatchString(`^\d+$`, executionNumberStr)
	return matched
}

func extractExecutionIdFromResponse(responseJSON string, targetExecutionName string) (string, error) {
	var executionGroups []map[string]any
	if err := json.Unmarshal([]byte(responseJSON), &executionGroups); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %v", err)
	}

	if len(executionGroups) == 0 {
		return "", fmt.Errorf("no execution found with name \"%s\"", targetExecutionName)
	}

	// Find matching execution
	matchingExecution := findMatchingExecution(executionGroups, targetExecutionName)
	if matchingExecution == nil {
		return "", fmt.Errorf("no execution ID found for \"%s\"", targetExecutionName)
	}

	if executionID, ok := matchingExecution["id"].(string); ok && executionID != "" {
		return executionID, nil
	}

	return "", fmt.Errorf("no execution ID found for \"%s\"", targetExecutionName)
}

func findMatchingExecution(executionGroups []map[string]any, targetExecutionName string) map[string]any {
	for _, group := range executionGroups {
		executions, ok := group["executions"].([]any)
		if !ok {
			continue
		}

		if len(executions) > 1 {
			// Find exact match by name
			for _, exec := range executions {
				if execution, ok := exec.(map[string]any); ok {
					if name, nameOk := execution["name"].(string); nameOk && name == targetExecutionName {
						return execution
					}
				}
			}
		}

		// If there's only one execution in the group, return it
		if len(executions) == 1 {
			if execution, ok := executions[0].(map[string]any); ok {
				return execution
			}
		}
	}

	return nil
}
