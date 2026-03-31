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
}
