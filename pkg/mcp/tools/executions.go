package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kubeshop/testkube/pkg/mcp/formatters"
)

type ExecutionLogger interface {
	GetExecutionLogs(ctx context.Context, executionId string) (string, error)
}

func FetchExecutionLogs(client ExecutionLogger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("fetch_execution_logs",
		mcp.WithDescription(FetchExecutionLogsDescription),
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
		mcp.WithDescription(ListExecutionsDescription),
		mcp.WithString("workflowName", mcp.Description(WorkflowNameDescription)),
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
		if pageStr := request.GetString("page", "0"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil && page >= 0 {
				params.Page = page
			}
		}

		result, err := client.ListExecutions(ctx, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list executions: %v", err)), nil
		}

		formatted, err := formatters.FormatListExecutions(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format executions: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

type ExecutionInfoGetter interface {
	GetExecutionInfo(ctx context.Context, workflowName, executionId string) (string, error)
}

func GetExecutionInfo(client ExecutionInfoGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_execution_info",
		mcp.WithDescription(GetExecutionInfoDescription),
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

		formatted, err := formatters.FormatExecutionInfo(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format execution info: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

type ExecutionLookup interface {
	LookupExecutionID(ctx context.Context, executionName string) (string, error)
}

func LookupExecutionId(client ExecutionLookup) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("lookup_execution_id",
		mcp.WithDescription(LookupExecutionIdDescription),
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
	var resultObject map[string]any
	if err := json.Unmarshal([]byte(responseJSON), &resultObject); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %v", err)
	}

	results, ok := resultObject["results"].([]any)
	if !ok || len(results) == 0 {
		return "", fmt.Errorf("no execution found with name \"%s\"", targetExecutionName)
	}

	for _, result := range results {
		if execution, ok := result.(map[string]any); ok {
			if name, nameOk := execution["name"].(string); nameOk && name == targetExecutionName {
				if executionID, idOk := execution["id"].(string); idOk && executionID != "" {
					return executionID, nil
				}
			}
		}
	}

	// Fallback to single result for backwards compatibility
	if len(results) == 1 {
		if execution, ok := results[0].(map[string]any); ok {
			if executionID, idOk := execution["id"].(string); idOk && executionID != "" {
				return executionID, nil
			}
		}
	}

	return "", fmt.Errorf("no execution ID found for \"%s\"", targetExecutionName)
}

type WorkflowExecutionAborter interface {
	AbortWorkflowExecution(ctx context.Context, workflowName, executionId string) (string, error)
}

func AbortWorkflowExecution(client WorkflowExecutionAborter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("abort_workflow_execution",
		mcp.WithDescription(AbortWorkflowExecutionDescription),
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

		result, err := client.AbortWorkflowExecution(ctx, workflowName, executionId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to abort workflow execution: %v", err)), nil
		}

		formatted, err := formatters.FormatAbortExecution(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format abort result: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

type ExecutionWaiter interface {
	WaitForExecutions(ctx context.Context, executionIds []string) (string, error)
}

func WaitForExecutions(client ExecutionWaiter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("wait_for_executions",
		mcp.WithDescription(WaitForExecutionsDescription),
		mcp.WithString("executionIds", mcp.Required(), mcp.Description(ExecutionIdsDescription)),
		mcp.WithString("timeoutMinutes", mcp.Description(TimeoutMinutesDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionIdsStr, err := RequiredParam[string](request, "executionIds")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Parse comma-separated execution IDs
		executionIds := strings.Split(executionIdsStr, ",")
		for i, id := range executionIds {
			executionIds[i] = strings.TrimSpace(id)
		}

		timeoutMinutes := 30 // default
		if timeoutStr := request.GetString("timeoutMinutes", ""); timeoutStr != "" {
			if timeout, err := strconv.Atoi(timeoutStr); err == nil && timeout > 0 {
				timeoutMinutes = timeout
			}
		}

		// Create a context with timeout
		if timeoutMinutes > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)
			defer cancel()
		}

		result, err := client.WaitForExecutions(ctx, executionIds)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to wait for executions: %v", err)), nil
		}

		formatted, err := formatters.FormatWaitForExecutions(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format wait results: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}
