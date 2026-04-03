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

// ExecutionLogParams holds optional filtering parameters for log retrieval.
// All fields are optional; zero values mean "no filter".
// When Tail, StartLine, and EndLine are all 0 the handler injects Tail=100
// so agents never receive an unbounded log response.
type ExecutionLogParams struct {
	Tail        int    // Return last N lines (0 → defaulted to 100 by the handler when no other range is set)
	StartLine   int    // 1-based start line (0 = from beginning)
	EndLine     int    // 1-based end line (0 = to end)
	Grep        string // Filter lines containing this substring
	Step        string // Filter to a specific workflow step by reference name
	WorkerRef   string // Worker instance ref from the 'workers' array in get_execution_info; when set, logs are fetched from the worker artifact instead of the main log
	WorkerIndex int    // 0-based worker index; only meaningful when WorkerRef is set (default 0)
}

type ExecutionLogger interface {
	GetExecutionLogs(ctx context.Context, executionId string, params ExecutionLogParams) (string, error)
}

func FetchExecutionLogs(client ExecutionLogger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("fetch_execution_logs",
		mcp.WithDescription(FetchExecutionLogsDescription),
		mcp.WithString("executionId",
			mcp.Required(),
			mcp.Description("The unique execution ID in MongoDB format (e.g., '67d2cdbc351aecb2720afdf2')."),
		),
		mcp.WithString("tail",
			mcp.Description("Return the last N lines of the log. Example: '50'"),
		),
		mcp.WithString("startLine",
			mcp.Description("1-based line number to start reading from. Use with endLine for a range."),
		),
		mcp.WithString("endLine",
			mcp.Description("1-based line number to stop reading at (inclusive). Use with startLine for a range."),
		),
		mcp.WithString("grep",
			mcp.Description("Filter to lines containing this substring (e.g., grep=ERROR)."),
		),
		mcp.WithString("step",
			mcp.Description("Filter to logs from a specific workflow step by reference name (e.g., 'run-tests', 'setup-env')."),
		),
		mcp.WithString("workerRef",
			mcp.Description("Worker instance ref from the 'workers' array returned by get_execution_info (e.g., 'r72qph9'). ONLY use values from that array — do NOT use step refs from the main log metadata. When set, fetches that specific worker's logs instead of the main execution log."),
		),
		mcp.WithString("workerIndex",
			mcp.Description("0-based index of the parallel worker to fetch logs from (default: 0). Use with workerRef."),
		),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionID := request.GetString("executionId", "")

		params := ExecutionLogParams{
			Grep:      request.GetString("grep", ""),
			Step:      request.GetString("step", ""),
			WorkerRef: request.GetString("workerRef", ""),
		}
		if tailStr := request.GetString("tail", ""); tailStr != "" {
			if v, err := strconv.Atoi(tailStr); err == nil && v > 0 {
				params.Tail = v
			}
		}
		if s := request.GetString("startLine", ""); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v > 0 {
				params.StartLine = v
			}
		}
		if s := request.GetString("endLine", ""); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v > 0 {
				params.EndLine = v
			}
		}
		if s := request.GetString("workerIndex", ""); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v >= 0 {
				params.WorkerIndex = v
			}
		}
		if params.StartLine > 0 && params.EndLine > 0 && params.StartLine > params.EndLine {
			return mcp.NewToolResultError(fmt.Sprintf("invalid line range: startLine (%d) must be less than or equal to endLine (%d)", params.StartLine, params.EndLine)), nil
		}

		// Default to the last 100 lines when no range restriction is given and grep is not
		// set. When grep is set the agent wants to search the full log; the server-side
		// match cap (100 results) bounds the response size instead.
		if params.Tail == 0 && params.StartLine == 0 && params.EndLine == 0 && params.Grep == "" {
			params.Tail = 100
		}

		logs, err := client.GetExecutionLogs(ctx, executionID, params)
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
	StartDate    string
	EndDate      string
}

type ExecutionLister interface {
	ListExecutions(ctx context.Context, params ListExecutionsParams) (string, error)
}

func ListExecutions(client ExecutionLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_executions",
		mcp.WithDescription(ListExecutionsDescription),
		mcp.WithString("workflowName", mcp.Description(WorkflowNameDescription)),
		mcp.WithString("selector", mcp.Description(SelectorDescription)),
		mcp.WithString("pageSize", mcp.Description(PageSizeDescription)),
		mcp.WithString("page", mcp.Description(PageDescription)),
		mcp.WithString("textSearch", mcp.Description(TextSearchDescription)),
		mcp.WithString("status", mcp.Description(StatusDescription)),
		mcp.WithString("since", mcp.Description(SinceDescription)),
		mcp.WithString("startDate", mcp.Description(StartDateDescription)),
		mcp.WithString("endDate", mcp.Description(EndDateDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := ListExecutionsParams{
			WorkflowName: request.GetString("workflowName", ""),
			Selector:     request.GetString("selector", ""),
			TextSearch:   request.GetString("textSearch", ""),
			Status:       request.GetString("status", ""),
			Since:        request.GetString("since", ""),
			StartDate:    request.GetString("startDate", ""),
			EndDate:      request.GetString("endDate", ""),
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
			return mcp.NewToolResultError(fmt.Sprintf("Invalid execution name format: \"%s\" expected format: \"workflow-name-number\".", executionName)), nil
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

type ExecutionTagUpdater interface {
	UpdateExecutionTags(ctx context.Context, executionId string, tags map[string]string) error
}

func UpdateExecutionTags(client ExecutionTagUpdater) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("update_execution_tags",
		mcp.WithDescription(UpdateExecutionTagsDescription),
		mcp.WithString("executionId", mcp.Required(), mcp.Description(ExecutionIdDescription)),
		mcp.WithObject("tags", mcp.Required(), mcp.Description(`Key-value tag pairs (e.g., {"env":"prod","bug":"found"}). Replaces all existing tags. Use {} to clear all tags.`)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionId, err := RequiredParam[string](request, "executionId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tagsRaw, ok, err := OptionalParamOK[map[string]any](request, "tags")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if !ok {
			return mcp.NewToolResultError("missing required parameter: tags"), nil
		}

		tags := make(map[string]string)
		for k, v := range tagsRaw {
			s, ok := v.(string)
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("tag value for key %q must be a string, got %T", k, v)), nil
			}
			tags[k] = s
		}

		if err := client.UpdateExecutionTags(ctx, executionId, tags); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update execution tags: %v", err)), nil
		}

		tagsJSON, err := json.Marshal(tags)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal tags: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Execution tags updated successfully. New tags: %s", tagsJSON)), nil
	}

	return tool, handler
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
