package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockInsightSeriesLister struct {
	result string
	err    error
	params InsightSeriesCatalogParams
}

func (m *mockInsightSeriesLister) ListInsightSeries(ctx context.Context, params InsightSeriesCatalogParams) (string, error) {
	m.params = params
	return m.result, m.err
}

type mockInsightMetricKeysLister struct {
	result string
	err    error
}

func (m *mockInsightMetricKeysLister) ListInsightMetricKeys(ctx context.Context, params InsightMetricKeysParams) (string, error) {
	return m.result, m.err
}

type mockInsightMetricSeriesGetter struct {
	result string
	err    error
	params InsightMetricSeriesParams
	called bool
}

func (m *mockInsightMetricSeriesGetter) GetInsightMetricSeries(ctx context.Context, params InsightMetricSeriesParams) (string, error) {
	m.called = true
	m.params = params
	return m.result, m.err
}

type mockInsightExecutionsLister struct {
	result string
	err    error
}

func (m *mockInsightExecutionsLister) ListInsightExecutions(ctx context.Context, params InsightExecutionsParams) (string, error) {
	return m.result, m.err
}

func TestListInsightSeries(t *testing.T) {
	t.Run("has expected name and passes filters", func(t *testing.T) {
		mock := &mockInsightSeriesLister{result: `{"items":[{"seriesId":"s1","workflowName":"wf","source":"k6","metricKey":"http_req_duration_p95_ms","identity":{}}],"hasMore":false}`}
		tool, handler := ListInsightSeries(mock)
		assert.Equal(t, "list_insight_series", tool.Name)

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{
			"workflow": "wf",
			"source":   "k6",
			"page":     "2",
		}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.False(t, result.IsError)
		assert.Contains(t, getResultText(result), "http_req_duration_p95_ms")
		assert.Equal(t, "wf", mock.params.Workflow)
		assert.Equal(t, "k6", mock.params.Source)
		assert.Equal(t, 2, mock.params.Page)
	})

	t.Run("empty catalog returns friendly message", func(t *testing.T) {
		mock := &mockInsightSeriesLister{result: `{"items":[],"hasMore":false}`}
		_, handler := ListInsightSeries(mock)
		result, err := handler(context.Background(), mcp.CallToolRequest{})
		require.NoError(t, err)
		assert.Contains(t, getResultText(result), "No insight series found")
	})

	t.Run("client error is surfaced", func(t *testing.T) {
		mock := &mockInsightSeriesLister{err: errors.New("boom")}
		_, handler := ListInsightSeries(mock)
		result, err := handler(context.Background(), mcp.CallToolRequest{})
		require.NoError(t, err)
		require.True(t, result.IsError)
		assert.Contains(t, getResultText(result), "boom")
	})
}

func TestListInsightMetricKeys(t *testing.T) {
	mock := &mockInsightMetricKeysLister{result: `{"items":["a","b"],"hasMore":true}`}
	tool, handler := ListInsightMetricKeys(mock)
	assert.Equal(t, "list_insight_metric_keys", tool.Name)

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	assert.Contains(t, getResultText(result), "metricKeys")
}

func TestGetInsightMetricSeries(t *testing.T) {
	t.Run("has expected name", func(t *testing.T) {
		tool, _ := GetInsightMetricSeries(&mockInsightMetricSeriesGetter{})
		assert.Equal(t, "get_insight_metric_series", tool.Name)
	})

	t.Run("requires measure or seriesId", func(t *testing.T) {
		mock := &mockInsightMetricSeriesGetter{}
		_, handler := GetInsightMetricSeries(mock)
		result, err := handler(context.Background(), mcp.CallToolRequest{})
		require.NoError(t, err)
		require.True(t, result.IsError)
		assert.Contains(t, getResultText(result), "either measure or seriesId is required")
		assert.False(t, mock.called, "client should not be called when both are missing")
	})

	t.Run("defaults aggregate to avg", func(t *testing.T) {
		mock := &mockInsightMetricSeriesGetter{result: `[{"ts":1,"value":10,"segments":[]}]`}
		_, handler := GetInsightMetricSeries(mock)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{"measure": "http_req_duration_p95_ms"}
		result, err := handler(context.Background(), request)
		require.NoError(t, err)
		require.False(t, result.IsError)
		assert.Equal(t, "avg", mock.params.Aggregate)
		assert.Contains(t, getResultText(result), "series")
	})

	t.Run("passes explicit aggregate through", func(t *testing.T) {
		mock := &mockInsightMetricSeriesGetter{result: `[{"ts":1,"value":10,"segments":[]}]`}
		_, handler := GetInsightMetricSeries(mock)
		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]any{"seriesId": "s1", "aggregate": "max"}
		_, err := handler(context.Background(), request)
		require.NoError(t, err)
		assert.Equal(t, "max", mock.params.Aggregate)
	})
}

func TestListInsightExecutions(t *testing.T) {
	// The control-plane endpoint returns a bare JSON array of execution refs
	// (pagination is carried in the Link header, not the body).
	mock := &mockInsightExecutionsLister{result: `[{"id":"e1","name":"n1","parent":"wf","status":"passed","duration":1200}]`}
	tool, handler := ListInsightExecutions(mock)
	assert.Equal(t, "list_insight_executions", tool.Name)

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	text := getResultText(result)
	assert.Contains(t, text, "e1")
	assert.Contains(t, text, "wf")
}
