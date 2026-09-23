package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/mcp/boards"
	"github.com/kubeshop/testkube/pkg/mcp/tools"
)

func TestAPIClient_Boards_RequestShapes(t *testing.T) {
	const (
		orgID = "org-1"
		envID = "env-1"
	)
	type seen struct {
		method, path string
		query        map[string]string
		body         map[string]any
	}
	var got seen
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = seen{method: r.Method, path: r.URL.Path, query: map[string]string{}}
		for k := range r.URL.Query() {
			got.query[k] = r.URL.Query().Get(k)
		}
		if data, _ := io.ReadAll(r.Body); len(data) > 0 {
			require.NoError(t, json.Unmarshal(data, &got.body))
		}
		if r.URL.Path == "/organizations/"+orgID+"/board-slug" {
			_, _ = io.WriteString(w, `{"available":true}`)
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	client := NewAPIClient(&MCPServerConfig{ControlPlaneUrl: server.URL, AccessToken: "user-token", OrgId: orgID, EnvId: envID}, server.Client())
	ctx := context.Background()
	base := "/organizations/" + orgID

	_, err := client.ListBoards(ctx, tools.ListBoardsParams{Name: "q", Shared: true, Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, seen{method: http.MethodGet, path: base + "/boards", query: map[string]string{"name": "q", "shared": "true", "page": "1", "pageSize": "20"}}, got)

	_, err = client.GetBoard(ctx, "my board")
	require.NoError(t, err)
	assert.Equal(t, base+"/boards/my board", got.path)

	available, err := client.CheckBoardSlug(ctx, "quality")
	require.NoError(t, err)
	assert.True(t, available)
	assert.Equal(t, "quality", got.query["slug"])

	_, err = client.CreateBoard(ctx, tools.CreateBoardParams{Name: "Q", IsPrivate: true})
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, got.method)
	assert.Equal(t, map[string]any{"name": "Q", "isPrivate": true}, got.body)

	description := "kept"
	_, err = client.UpdateBoard(ctx, "tkcbrd_1", tools.UpdateBoardRequest{
		Description: &description,
		Content: &tools.BoardContentPatch{Action: "create", ContentKind: "report", ContentData: &boards.ReportDraft{
			Kind: "workflows", Name: "W", Params: map[string]any{"duration": "month"},
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, http.MethodPatch, got.method)
	assert.Equal(t, base+"/boards/tkcbrd_1", got.path)
	assert.Equal(t, map[string]any{
		"description": "kept",
		"content": map[string]any{
			"action": "create", "content_kind": "report",
			"content_data": map[string]any{"kind": "workflows", "name": "W", "params": map[string]any{"duration": "month"}},
		},
	}, got.body, "no layout key is sent when the layout is not changed")

	require.NoError(t, client.DeleteBoard(ctx, "tkcbrd_1"))
	assert.Equal(t, http.MethodDelete, got.method)

	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	query := boards.InsightQuery{Endpoint: boards.EndpointWorkflows, StartDate: start, EndDate: start.AddDate(0, 0, 7)}
	_, err = client.QueryBoardInsights(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, base+"/insights/workflows", got.path)
	assert.NotContains(t, got.query, "env", "a report without an environment filter covers the whole organization")
	assert.Equal(t, "2026-09-01T00:00:00Z", got.query["startDate"])

	query.CurrentEnvironment = true
	_, err = client.QueryBoardInsights(ctx, query)
	require.NoError(t, err)
	assert.Equal(t, envID, got.query["env"])
}

func TestAPIClient_Boards_RefuseAPITokenWithoutRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()

	client := NewAPIClient(&MCPServerConfig{ControlPlaneUrl: server.URL, AccessToken: "tkcapi_abc", OrgId: "o", EnvId: "e"}, server.Client())
	ctx := context.Background()

	_, err := client.ListBoards(ctx, tools.ListBoardsParams{})
	assert.True(t, errors.Is(err, tools.ErrBoardsRequireUser))
	_, err = client.GetBoard(ctx, "b")
	assert.True(t, errors.Is(err, tools.ErrBoardsRequireUser))
	_, err = client.CheckBoardSlug(ctx, "s")
	assert.True(t, errors.Is(err, tools.ErrBoardsRequireUser))
	_, err = client.CreateBoard(ctx, tools.CreateBoardParams{Name: "n"})
	assert.True(t, errors.Is(err, tools.ErrBoardsRequireUser))
	_, err = client.UpdateBoard(ctx, "b", tools.UpdateBoardRequest{})
	assert.True(t, errors.Is(err, tools.ErrBoardsRequireUser))
	assert.True(t, errors.Is(client.DeleteBoard(ctx, "b"), tools.ErrBoardsRequireUser))
	assert.Zero(t, requests.Load(), "an API token must be refused before any request is sent")
}
