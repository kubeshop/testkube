package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDashboardUrl(t *testing.T) {
	const (
		dashboardUrl = "https://app.testkube.io"
		orgId        = "org-123"
		envId        = "env-456"
	)

	basePath := "/organization/" + orgId + "/environment/" + envId + "/dashboard"

	parseURL := func(t *testing.T, result *mcp.CallToolResult) string {
		t.Helper()
		require.NotNil(t, result)
		require.NotEmpty(t, result.Content)
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok, "expected TextContent")
		var parsed map[string]string
		err := json.Unmarshal([]byte(textContent.Text), &parsed)
		require.NoError(t, err)
		return parsed["url"]
	}

	t.Run("tool has correct name", func(t *testing.T) {
		tool, _ := BuildDashboardUrl(dashboardUrl, orgId, envId)
		assert.Equal(t, "build_dashboard_url", tool.Name)
	})

	t.Run("workflow URL without execution", func(t *testing.T) {
		_, handler := BuildDashboardUrl(dashboardUrl, orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "workflow",
			"workflowName": "my-wf",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		url := parseURL(t, result)
		assert.Equal(t, dashboardUrl+basePath+"/test-workflows/my-wf", url)
	})

	t.Run("workflow URL with executionId", func(t *testing.T) {
		_, handler := BuildDashboardUrl(dashboardUrl, orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "workflow",
			"workflowName": "my-wf",
			"executionId":  "exec-1",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		url := parseURL(t, result)
		assert.Equal(t, dashboardUrl+basePath+"/test-workflows/my-wf/execution/exec-1", url)
	})

	t.Run("execution URL baseline", func(t *testing.T) {
		_, handler := BuildDashboardUrl(dashboardUrl, orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "execution",
			"workflowName": "my-wf",
			"executionId":  "exec-1",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		url := parseURL(t, result)
		assert.Equal(t, dashboardUrl+basePath+"/test-workflows/my-wf/execution/exec-1", url)
	})

	t.Run("execution URL with stepRef defaults tab to log-output", func(t *testing.T) {
		_, handler := BuildDashboardUrl(dashboardUrl, orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "execution",
			"workflowName": "my-wf",
			"executionId":  "exec-1",
			"stepRef":      "rwhc2zn",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		url := parseURL(t, result)
		assert.Equal(t, dashboardUrl+basePath+"/test-workflows/my-wf/execution/exec-1/log-output?ref=rwhc2zn", url)
	})

	t.Run("execution URL with stepRef and explicit executionTab", func(t *testing.T) {
		_, handler := BuildDashboardUrl(dashboardUrl, orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "execution",
			"workflowName": "my-wf",
			"executionId":  "exec-1",
			"stepRef":      "rwhc2zn",
			"executionTab": "artifacts",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		url := parseURL(t, result)
		assert.Equal(t, dashboardUrl+basePath+"/test-workflows/my-wf/execution/exec-1/artifacts?ref=rwhc2zn", url)
	})

	t.Run("execution URL with executionTab only no query param", func(t *testing.T) {
		_, handler := BuildDashboardUrl(dashboardUrl, orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "execution",
			"workflowName": "my-wf",
			"executionId":  "exec-1",
			"executionTab": "log-output",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		url := parseURL(t, result)
		assert.Equal(t, dashboardUrl+basePath+"/test-workflows/my-wf/execution/exec-1/log-output", url)
	})

	t.Run("stepRef without executionId returns error", func(t *testing.T) {
		_, handler := BuildDashboardUrl(dashboardUrl, orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "workflow",
			"workflowName": "my-wf",
			"stepRef":      "rwhc2zn",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})

	t.Run("workflow URL with executionId and stepRef", func(t *testing.T) {
		_, handler := BuildDashboardUrl(dashboardUrl, orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "workflow",
			"workflowName": "my-wf",
			"executionId":  "exec-1",
			"stepRef":      "abc123",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		url := parseURL(t, result)
		assert.Equal(t, dashboardUrl+basePath+"/test-workflows/my-wf/execution/exec-1/log-output?ref=abc123", url)
	})

	t.Run("missing dashboardUrl returns error", func(t *testing.T) {
		_, handler := BuildDashboardUrl("", orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "workflow",
			"workflowName": "my-wf",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})

	t.Run("missing workflowName returns error", func(t *testing.T) {
		_, handler := BuildDashboardUrl(dashboardUrl, orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "execution",
			"executionId":  "exec-1",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})

	t.Run("unsupported resourceType returns error", func(t *testing.T) {
		_, handler := BuildDashboardUrl(dashboardUrl, orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "unknown",
			"workflowName": "my-wf",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})

	t.Run("execution without executionId returns error", func(t *testing.T) {
		_, handler := BuildDashboardUrl(dashboardUrl, orgId, envId)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"resourceType": "execution",
			"workflowName": "my-wf",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.IsError)
	})
}
