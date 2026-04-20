package tools

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutionLogger records the params passed to GetExecutionLogs.
type mockExecutionLogger struct {
	capturedID     string
	capturedParams ExecutionLogParams
	returnLogs     string
	returnErr      error
}

func (m *mockExecutionLogger) GetExecutionLogs(_ context.Context, id string, params ExecutionLogParams) (string, error) {
	m.capturedID = id
	m.capturedParams = params
	return m.returnLogs, m.returnErr
}

func callFetchExecutionLogs(t *testing.T, mock *mockExecutionLogger, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()
	_, handler := FetchExecutionLogs(mock)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return handler(context.Background(), req)
}

func TestFetchExecutionLogs_NoParams_DefaultsTail100(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "log output"}
	result, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "abc123",
	})
	require.NoError(t, err)
	assert.Equal(t, "abc123", m.capturedID)
	// No range params → handler must inject Tail=100 so agents never get unbounded logs.
	assert.Equal(t, ExecutionLogParams{Tail: 100}, m.capturedParams)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, "log output", textContent.Text)
}

func TestFetchExecutionLogs_AllParams_ParsedCorrectly(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "filtered logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "def456",
		"tail":        "50",
		"startLine":   "100",
		"endLine":     "200",
		"grep":        "ERROR",
		"step":        "run-tests",
	})
	require.NoError(t, err)
	assert.Equal(t, "def456", m.capturedID)
	assert.Equal(t, ExecutionLogParams{
		Tail:      50,
		StartLine: 100,
		EndLine:   200,
		Grep:      "ERROR",
		Step:      "run-tests",
	}, m.capturedParams)
}

func TestFetchExecutionLogs_GrepOnly_NoTailInjected(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "grep results"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "grep001",
		"grep":        "ERROR",
	})
	require.NoError(t, err)
	// grep searches the full log — Tail must NOT be injected so the whole log is scanned.
	// Server-side match cap bounds the output size instead.
	assert.Equal(t, ExecutionLogParams{Grep: "ERROR"}, m.capturedParams)
}

func TestFetchExecutionLogs_StepOnly_TailInjected(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "step logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "step001",
		"step":        "run-tests",
	})
	require.NoError(t, err)
	// step with no range/grep → default tail=100 applied to that step's lines.
	assert.Equal(t, ExecutionLogParams{Tail: 100, Step: "run-tests"}, m.capturedParams)
}

func TestFetchExecutionLogs_GrepAndStep_ParsedCorrectly(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "step logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "ghi789",
		"grep":        "FAIL",
		"step":        "setup-env",
	})
	require.NoError(t, err)
	// grep is set → no Tail injection; server-side match cap handles output size.
	assert.Equal(t, ExecutionLogParams{
		Grep: "FAIL",
		Step: "setup-env",
	}, m.capturedParams)
}

func TestFetchExecutionLogs_InvalidIntParams_Ignored(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "jkl012",
		"tail":        "not-a-number",
		"startLine":   "-5",
		"endLine":     "0",
	})
	require.NoError(t, err)
	// "not-a-number" fails Atoi; "-5" parses as -5 which fails the v>0 guard; "0" also fails v>0.
	// All integer params are discarded → no range set → handler injects Tail=100.
	assert.Equal(t, ExecutionLogParams{Tail: 100}, m.capturedParams)
}

func TestFetchExecutionLogs_TailZero_FallsBackToDefault(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "mno345",
		"tail":        "0",
	})
	require.NoError(t, err)
	// tail=0 is rejected by the v>0 guard; no other range set → default Tail=100 applied.
	assert.Equal(t, 100, m.capturedParams.Tail)
}

func TestFetchExecutionLogs_InvalidLineRange_ReturnsError(t *testing.T) {
	m := &mockExecutionLogger{}
	result, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "pqr678",
		"startLine":   "200",
		"endLine":     "100",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError, "expected tool result error for startLine > endLine")
	// Client must not have been called
	assert.Equal(t, "", m.capturedID)
}

func TestFetchExecutionLogs_ValidLineRange_Passes(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "stu901",
		"startLine":   "100",
		"endLine":     "200",
	})
	require.NoError(t, err)
	assert.Equal(t, 100, m.capturedParams.StartLine)
	assert.Equal(t, 200, m.capturedParams.EndLine)
}

func TestFetchExecutionLogs_EqualLineRange_Passes(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "vwx234",
		"startLine":   "100",
		"endLine":     "100",
	})
	require.NoError(t, err)
	assert.Equal(t, 100, m.capturedParams.StartLine)
	assert.Equal(t, 100, m.capturedParams.EndLine)
}

func TestFetchExecutionLogs_ToolName(t *testing.T) {
	m := &mockExecutionLogger{}
	tool, _ := FetchExecutionLogs(m)
	assert.Equal(t, "fetch_execution_logs", tool.Name)
}

func TestFetchExecutionLogs_ToolHasExpectedParams(t *testing.T) {
	m := &mockExecutionLogger{}
	tool, _ := FetchExecutionLogs(m)
	paramNames := make([]string, 0)
	for name := range tool.InputSchema.Properties {
		paramNames = append(paramNames, name)
	}
	assert.Contains(t, paramNames, "executionId")
	assert.Contains(t, paramNames, "tail")
	assert.Contains(t, paramNames, "startLine")
	assert.Contains(t, paramNames, "endLine")
	assert.Contains(t, paramNames, "grep")
	assert.Contains(t, paramNames, "step")
	assert.Contains(t, paramNames, "workerRef")
	assert.Contains(t, paramNames, "workerIndex")
}

func TestFetchExecutionLogs_WorkerRefAndIndex_ParsedCorrectly(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "worker logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "abc123",
		"workerRef":   "r72qph9",
		"workerIndex": "2",
	})
	require.NoError(t, err)
	assert.Equal(t, "r72qph9", m.capturedParams.WorkerRef)
	assert.Equal(t, 2, m.capturedParams.WorkerIndex)
	// No other range params → default tail=100 still applies.
	assert.Equal(t, 100, m.capturedParams.Tail)
}

func TestFetchExecutionLogs_WorkerRefOnly_DefaultsIndex0(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "worker logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "abc123",
		"workerRef":   "r72qph9",
	})
	require.NoError(t, err)
	assert.Equal(t, "r72qph9", m.capturedParams.WorkerRef)
	assert.Equal(t, 0, m.capturedParams.WorkerIndex)
}

func TestFetchExecutionLogs_WorkerIndexWithoutRef_Parsed(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "abc123",
		"workerIndex": "5",
	})
	require.NoError(t, err)
	// WorkerRef is empty; WorkerIndex is parsed but the server ignores it without a ref.
	assert.Equal(t, "", m.capturedParams.WorkerRef)
	assert.Equal(t, 5, m.capturedParams.WorkerIndex)
}

func TestFetchExecutionLogs_WorkerWithGrep_Combined(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "filtered worker logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "abc123",
		"workerRef":   "r72qph9",
		"workerIndex": "1",
		"grep":        "ERROR",
	})
	require.NoError(t, err)
	assert.Equal(t, "r72qph9", m.capturedParams.WorkerRef)
	assert.Equal(t, 1, m.capturedParams.WorkerIndex)
	assert.Equal(t, "ERROR", m.capturedParams.Grep)
	// grep set → no tail injection.
	assert.Equal(t, 0, m.capturedParams.Tail)
}

func TestFetchExecutionLogs_InvalidWorkerIndex_FallsBackToZero(t *testing.T) {
	m := &mockExecutionLogger{returnLogs: "logs"}
	_, err := callFetchExecutionLogs(t, m, map[string]any{
		"executionId": "abc123",
		"workerRef":   "r72qph9",
		"workerIndex": "not-a-number",
	})
	require.NoError(t, err)
	assert.Equal(t, "r72qph9", m.capturedParams.WorkerRef)
	// Invalid index falls back to zero (zero value of int).
	assert.Equal(t, 0, m.capturedParams.WorkerIndex)
}

// mockExecutionLister records the params passed to ListExecutions.
type mockExecutionLister struct {
	capturedParams ListExecutionsParams
	returnResult   string
	returnErr      error
}

func (m *mockExecutionLister) ListExecutions(_ context.Context, params ListExecutionsParams) (string, error) {
	m.capturedParams = params
	return m.returnResult, m.returnErr
}

func TestListExecutions_ToolHasExpectedParams(t *testing.T) {
	m := &mockExecutionLister{}
	tool, _ := ListExecutions(m)
	assert.Equal(t, "list_executions", tool.Name)

	paramNames := make([]string, 0)
	for name := range tool.InputSchema.Properties {
		paramNames = append(paramNames, name)
	}
	assert.Contains(t, paramNames, "workflowName")
	assert.Contains(t, paramNames, "selector")
	assert.Contains(t, paramNames, "tagSelector")
	assert.Contains(t, paramNames, "status")
	assert.Contains(t, paramNames, "since")
	assert.Contains(t, paramNames, "startDate")
	assert.Contains(t, paramNames, "endDate")
	assert.Contains(t, paramNames, "pageSize")
	assert.Contains(t, paramNames, "page")
	assert.Contains(t, paramNames, "textSearch")
}

func TestListExecutions_TagSelectorPassedThrough(t *testing.T) {
	m := &mockExecutionLister{returnResult: `{"totals":{"results":0},"results":[]}`}
	_, handler := ListExecutions(m)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"tagSelector": "type=suite,env=prod",
		"since":       "2026-03-31T09:00:00Z",
		"selector":    "tool=cypress",
	}
	_, err := handler(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "type=suite,env=prod", m.capturedParams.TagSelector)
	assert.Equal(t, "2026-03-31T09:00:00Z", m.capturedParams.Since)
	assert.Equal(t, "tool=cypress", m.capturedParams.Selector)
}

// mockExecutionBulkGetter records the params passed to GetExecutions.
type mockExecutionBulkGetter struct {
	capturedParams ListExecutionsParams
	returnResult   map[string]string
	returnErr      error
}

func (m *mockExecutionBulkGetter) GetExecutions(_ context.Context, params ListExecutionsParams) (map[string]string, error) {
	m.capturedParams = params
	return m.returnResult, m.returnErr
}

func TestQueryExecutions_ToolHasExpectedParams(t *testing.T) {
	m := &mockExecutionBulkGetter{}
	tool, _ := QueryExecutions(m)
	assert.Equal(t, "query_executions", tool.Name)

	paramNames := make([]string, 0)
	for name := range tool.InputSchema.Properties {
		paramNames = append(paramNames, name)
	}
	assert.Contains(t, paramNames, "expression")
	assert.Contains(t, paramNames, "workflowName")
	assert.Contains(t, paramNames, "status")
	assert.Contains(t, paramNames, "selector")
	assert.Contains(t, paramNames, "tagSelector")
	assert.Contains(t, paramNames, "since")
	assert.Contains(t, paramNames, "startDate")
	assert.Contains(t, paramNames, "endDate")
	assert.Contains(t, paramNames, "limit")
	assert.Contains(t, paramNames, "aggregate")
}

func TestQueryExecutions_FiltersPassedThrough(t *testing.T) {
	m := &mockExecutionBulkGetter{returnResult: map[string]string{
		"exec-1": `{"id":"exec-1","result":{"status":"passed","duration":"45s"}}`,
	}}
	_, handler := QueryExecutions(m)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"expression":  "$.result.status",
		"tagSelector": "type=suite",
		"selector":    "tool=k6",
		"since":       "2026-03-31T09:00:00Z",
		"startDate":   "2026-03-31",
		"endDate":     "2026-03-31",
		"status":      "passed",
	}
	_, err := handler(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "type=suite", m.capturedParams.TagSelector)
	assert.Equal(t, "tool=k6", m.capturedParams.Selector)
	assert.Equal(t, "2026-03-31T09:00:00Z", m.capturedParams.Since)
	assert.Equal(t, "2026-03-31", m.capturedParams.StartDate)
	assert.Equal(t, "2026-03-31", m.capturedParams.EndDate)
	assert.Equal(t, "passed", m.capturedParams.Status)
}
