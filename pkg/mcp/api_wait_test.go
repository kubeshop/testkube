package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/mcp/tools"
)

// The wait_for_executions tool must report each finished run's id, name and
// status, so an MCP client can learn the outcome of the run it triggered; the
// formatter reads them from TestWorkflowExecution records.
func TestAPIClient_WaitForExecutions_ToolReportsOutcome(t *testing.T) {
	const (
		orgID = "org-1"
		envID = "env-1"
	)
	executions := map[string]testkube.TestWorkflowExecution{}
	for _, e := range []struct {
		id, name string
		status   testkube.TestWorkflowStatus
	}{
		{"exec-passed", "smoke-1", testkube.PASSED_TestWorkflowStatus},
		{"exec-failed", "smoke-2", testkube.FAILED_TestWorkflowStatus},
	} {
		executions[e.id] = testkube.TestWorkflowExecution{
			Id:     e.id,
			Name:   e.name,
			Result: &testkube.TestWorkflowResult{Status: &e.status},
		}
	}

	prefix := "/organizations/" + orgID + "/environments/" + envID + "/agent/test-workflow-executions/"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exec, ok := executions[strings.TrimPrefix(r.URL.Path, prefix)]
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, prefix) || !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(exec))
	}))
	defer server.Close()

	client := NewAPIClient(&MCPServerConfig{
		ControlPlaneUrl: server.URL,
		AccessToken:     "tok",
		OrgId:           orgID,
		EnvId:           envID,
	}, server.Client())
	client.waitPollInterval = time.Millisecond

	_, handler := tools.WaitForExecutions(client)
	req := mcp.CallToolRequest{}
	req.Params.Name = "wait_for_executions"
	req.Params.Arguments = map[string]any{"executionIds": "exec-passed, exec-failed"}
	res, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.False(t, res.IsError, "tool error: %v", res.Content)

	var got struct {
		Executions []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"executions"`
	}
	text := res.Content[0].(mcp.TextContent).Text
	require.NoError(t, json.Unmarshal([]byte(text), &got), text)
	require.Len(t, got.Executions, 2, text)
	assert.Equal(t, "exec-passed", got.Executions[0].ID, text)
	assert.Equal(t, "smoke-1", got.Executions[0].Name, text)
	assert.Equal(t, "passed", got.Executions[0].Status, text)
	assert.Equal(t, "exec-failed", got.Executions[1].ID, text)
	assert.Equal(t, "smoke-2", got.Executions[1].Name, text)
	assert.Equal(t, "failed", got.Executions[1].Status, text)
}
