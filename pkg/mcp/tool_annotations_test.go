package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Hints as {readOnly, destructive, idempotent, openWorld}.
var (
	readHints        = [4]bool{true, false, true, false}
	additiveHints    = [4]bool{false, false, false, false}
	destructiveHints = [4]bool{false, true, true, false}
	runHints         = [4]bool{false, true, false, true}
)

// expectedToolHints lists every registered tool; a new tool fails the test until it is added here.
var expectedToolHints = map[string][4]bool{
	"build_dashboard_url":             readHints,
	"list_workflows":                  readHints,
	"get_workflow":                    readHints,
	"get_workflow_definition":         readHints,
	"get_workflow_metrics":            readHints,
	"list_workflowtemplates":          readHints,
	"get_workflowtemplate_definition": readHints,
	"query_workflows":                 readHints,
	"query_executions":                readHints,
	"get_workflow_schema":             readHints,
	"get_execution_schema":            readHints,
	"list_labels":                     readHints,
	"list_resource_groups":            readHints,
	"list_agents":                     readHints,
	"fetch_execution_logs":            readHints,
	"list_executions":                 readHints,
	"lookup_execution_id":             readHints,
	"get_execution_info":              readHints,
	"get_workflow_execution_metrics":  readHints,
	"get_workflow_resource_history":   readHints,
	"wait_for_executions":             readHints,
	"list_artifacts":                  readHints,
	"read_artifact":                   readHints,
	"list_insight_series":             readHints,
	"list_insight_metric_keys":        readHints,
	"get_insight_metric_series":       readHints,
	"list_insight_executions":         readHints,
	"list_boards":                     readHints,
	"get_board":                       readHints,
	"render_board":                    readHints,
	"create_workflow":                 additiveHints,
	"create_workflowtemplate":         additiveHints,
	"create_board":                    additiveHints,
	"add_board_report":                additiveHints,
	"update_workflow":                 destructiveHints,
	"update_workflowtemplate":         destructiveHints,
	"update_execution_tags":           destructiveHints,
	"abort_workflow_execution":        destructiveHints,
	"update_board":                    destructiveHints,
	"update_board_report":             destructiveHints,
	"remove_board_report":             destructiveHints,
	"delete_board":                    destructiveHints,
	"run_workflow":                    runHints,
}

type listedTool struct {
	Name        string `json:"name"`
	Annotations struct {
		ReadOnly    *bool `json:"readOnlyHint"`
		Destructive *bool `json:"destructiveHint"`
		Idempotent  *bool `json:"idempotentHint"`
		OpenWorld   *bool `json:"openWorldHint"`
	} `json:"annotations"`
}

func TestToolAnnotations(t *testing.T) {
	tools := listServerTools(t)

	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	expected := make([]string, 0, len(expectedToolHints))
	for name := range expectedToolHints {
		expected = append(expected, name)
	}
	slices.Sort(names)
	slices.Sort(expected)
	require.Equal(t, expected, names, "every registered tool needs an entry in expectedToolHints")

	for _, tool := range tools {
		a := tool.Annotations
		require.NotNil(t, a.ReadOnly, tool.Name)
		require.NotNil(t, a.Destructive, tool.Name)
		require.NotNil(t, a.Idempotent, tool.Name)
		require.NotNil(t, a.OpenWorld, tool.Name)
		got := [4]bool{*a.ReadOnly, *a.Destructive, *a.Idempotent, *a.OpenWorld}
		assert.Equal(t, expectedToolHints[tool.Name], got, tool.Name)
	}
}

// listServerTools returns tools/list as a client sees it over Streamable HTTP.
func listServerTools(t *testing.T) []listedTool {
	t.Helper()
	_, srv, err := newSHTTPServer(shttpTestConfig(), nil)
	require.NoError(t, err)
	endpoint := httptest.NewServer(srv.Handler)
	t.Cleanup(endpoint.Close)
	client := endpoint.Client()
	client.Timeout = 3 * time.Second

	post := func(message, session string) (*http.Response, []byte) {
		req, err := http.NewRequestWithContext(t.Context(), "POST", endpoint.URL+"/mcp", strings.NewReader(message))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", "Bearer inbound-token")
		if session != "" {
			req.Header.Set("Mcp-Session-Id", session)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
		return resp, body
	}

	resp, _ := post(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`, "")
	session := resp.Header.Get("Mcp-Session-Id")
	require.NotEmpty(t, session)

	_, body := post(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, session)
	var result struct {
		Result struct {
			Tools []listedTool `json:"tools"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(body, &result))
	require.NotEmpty(t, result.Result.Tools)
	return result.Result.Tools
}
