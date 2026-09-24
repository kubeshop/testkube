package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/mcp/boards"
)

const testBoard = `{
	"id": "tkcbrd_1", "slug": "quality", "name": "Quality", "description": "Keep me", "shared": true,
	"layout": {"version": 1, "rows": [{"id": "r1", "cells": [{"id": "aa"}, {"id": "bb"}]}, {"id": "r2", "cells": [{"id": "cc"}]}]},
	"content": {"reports": [
		{"id": "aa", "kind": "pass-fail", "name": "P/F", "params": {"measure": "ratio", "duration": "month", "filter": []}},
		{"id": "bb", "kind": "executions", "name": "Exec", "params": {"groupBy": "workflow", "measure": "count", "filter": [
			{"filterConfigurationKey": "workflow", "id": "w", "operator": "is", "value": ["api"]}]}},
		{"id": "cc", "kind": "time-series", "name": "TS", "params": {"measure": "", "aggregate": "sum", "filter": []}}
	]}
}`

// fakeBoardClient records the requests the board tools make.
type fakeBoardClient struct {
	mu sync.Mutex

	board     string
	getErr    error
	updateErr error
	// updated is returned by UpdateBoard; defaults to board.
	updated string

	slugAvailable bool
	listParams    ListBoardsParams
	created       *CreateBoardParams
	updates       []UpdateBoardRequest
	updatedRef    string
	deleted       string
	queries       []boards.InsightQuery
	queryResult   map[boards.Endpoint]string
	queryErr      map[boards.Endpoint]error

	// boardReads are the boards later reads return, in order.
	boardReads []string
	// updateErrs are the errors successive updates return, in order.
	updateErrs []error
}

func (f *fakeBoardClient) ListBoards(_ context.Context, p ListBoardsParams) (string, error) {
	f.listParams = p
	return `[{"id":"tkcbrd_1","slug":"quality","name":"Quality","shared":true,"analysisCount":3}]`, nil
}

func (f *fakeBoardClient) GetBoard(context.Context, string) (string, error) {
	// Each read after the first returns the next board of boardReads, as if
	// someone edited the board in between.
	if len(f.boardReads) > 0 {
		f.board, f.boardReads = f.boardReads[0], f.boardReads[1:]
	}
	return f.board, f.getErr
}

func (f *fakeBoardClient) CheckBoardSlug(context.Context, string) (bool, error) {
	return f.slugAvailable, nil
}

func (f *fakeBoardClient) CreateBoard(_ context.Context, p CreateBoardParams) (string, error) {
	f.created = &p
	return f.board, nil
}

func (f *fakeBoardClient) UpdateBoard(_ context.Context, ref string, r UpdateBoardRequest) (string, error) {
	f.updatedRef = ref
	f.updates = append(f.updates, r)
	if len(f.updateErrs) > 0 {
		err := f.updateErrs[0]
		f.updateErrs = f.updateErrs[1:]
		if err != nil {
			return "", err
		}
	}
	if f.updated != "" {
		return f.updated, f.updateErr
	}
	return f.board, f.updateErr
}

func (f *fakeBoardClient) DeleteBoard(_ context.Context, ref string) error {
	f.deleted = ref
	return nil
}

func (f *fakeBoardClient) QueryBoardInsights(_ context.Context, q boards.InsightQuery) (string, error) {
	f.mu.Lock()
	f.queries = append(f.queries, q)
	f.mu.Unlock()
	if err := f.queryErr[q.Endpoint]; err != nil {
		return "", err
	}
	return f.queryResult[q.Endpoint], nil
}

func callTool(t *testing.T, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) *mcp.CallToolResult {
	t.Helper()
	request := mcp.CallToolRequest{}
	request.Params.Arguments = args
	result, err := handler(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)
	return result
}

func (f *fakeBoardClient) lastUpdate(t *testing.T) UpdateBoardRequest {
	t.Helper()
	require.NotEmpty(t, f.updates, "expected an update request")
	return f.updates[len(f.updates)-1]
}

func TestListBoards(t *testing.T) {
	f := &fakeBoardClient{}
	tool, handler := ListBoards(f)
	assert.Equal(t, "list_boards", tool.Name)
	require.NotNil(t, tool.Annotations.ReadOnlyHint)
	assert.True(t, *tool.Annotations.ReadOnlyHint)

	result := callTool(t, handler, map[string]any{"visibility": "private", "favorite": "user", "pageSize": float64(1)})
	require.False(t, result.IsError, getResultText(result))
	assert.Equal(t, ListBoardsParams{Private: true, UserFavorite: true, PageSize: 1}, f.listParams)
	assert.Contains(t, getResultText(result), `"hasMore":true`)
	assert.Contains(t, getResultText(result), `"reports":3`)

	result = callTool(t, handler, map[string]any{"visibility": "team"})
	assert.True(t, result.IsError)
}

func TestGetBoard_ReportsInLayoutOrder(t *testing.T) {
	f := &fakeBoardClient{board: testBoard}
	_, handler := GetBoard(f)
	result := callTool(t, handler, map[string]any{"board": "quality"})
	require.False(t, result.IsError, getResultText(result))
	text := getResultText(result)
	assert.Contains(t, text, `"layout":[["aa","bb"],["cc"]]`)
	assert.Less(t, strings.Index(text, `"id":"aa"`), strings.Index(text, `"id":"cc"`))
}

func TestBoardTools_TokenRejection(t *testing.T) {
	f := &fakeBoardClient{getErr: ErrBoardsRequireUser}
	_, handler := GetBoard(f)
	result := callTool(t, handler, map[string]any{"board": "quality"})
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "testkube login")

	// The Control Plane's own refusal, as a newer version reports it.
	f.getErr = errors.New("API returned status 403: boards with API tokens are not supported")
	result = callTool(t, handler, map[string]any{"board": "quality"})
	assert.Contains(t, getResultText(result), "testkube login")
}

func TestCreateBoard(t *testing.T) {
	t.Run("refuses a taken slug", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard, slugAvailable: false}
		_, handler := CreateBoard(f)
		result := callTool(t, handler, map[string]any{"name": "Quality", "slug": "quality"})
		assert.True(t, result.IsError)
		assert.Contains(t, getResultText(result), "already used")
		assert.Nil(t, f.created)
	})

	t.Run("creates", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard, slugAvailable: true}
		_, handler := CreateBoard(f)
		result := callTool(t, handler, map[string]any{"name": "Quality", "slug": "quality", "private": true, "description": "d"})
		require.False(t, result.IsError, getResultText(result))
		assert.Equal(t, &CreateBoardParams{Name: "Quality", Slug: "quality", Description: "d", IsPrivate: true}, f.created)
	})
}

func TestUpdateBoard(t *testing.T) {
	t.Run("keeps the description when only the name changes", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard}
		_, handler := UpdateBoard(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "name": "Renamed"})
		require.False(t, result.IsError, getResultText(result))
		u := f.lastUpdate(t)
		require.NotNil(t, u.Name)
		assert.Equal(t, "Renamed", *u.Name)
		require.NotNil(t, u.Description)
		assert.Equal(t, "Keep me", *u.Description)
		assert.Equal(t, "tkcbrd_1", f.updatedRef, "the update addresses the board by ID even when given a slug")
	})

	t.Run("clears the description on request", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard}
		_, handler := UpdateBoard(f)
		callTool(t, handler, map[string]any{"board": "quality", "description": ""})
		assert.Equal(t, "", *f.lastUpdate(t).Description)
	})

	t.Run("validates the layout", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard}
		_, handler := UpdateBoard(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "layout": map[string]any{
			"version": float64(1), "rows": []any{map[string]any{"cells": []any{map[string]any{"id": "aa"}}}},
		}})
		assert.True(t, result.IsError)
		assert.Contains(t, getResultText(result), "leaves out reports bb, cc")
		assert.Empty(t, f.updates)

		result = callTool(t, handler, map[string]any{"board": "quality", "layout": map[string]any{
			"version": float64(1), "rows": []any{
				map[string]any{"cells": []any{map[string]any{"id": "cc"}}},
				map[string]any{"cells": []any{map[string]any{"id": "aa"}, map[string]any{"id": "bb"}}},
			},
		}})
		require.False(t, result.IsError, getResultText(result))
		var layout boards.Layout
		require.NoError(t, json.Unmarshal(f.lastUpdate(t).Layout, &layout))
		assert.Equal(t, [][]string{{"cc"}, {"aa", "bb"}}, layout.RowIDs())
	})

	t.Run("refuses an empty update", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard}
		_, handler := UpdateBoard(f)
		result := callTool(t, handler, map[string]any{"board": "quality"})
		assert.True(t, result.IsError)
		assert.Empty(t, f.updates)
	})
}

func TestAddBoardReport(t *testing.T) {
	t.Run("normalizes params, applies filters and returns the new report ID", func(t *testing.T) {
		updated := strings.Replace(testBoard, `{"id": "aa",`, `{"id": "dd", "kind": "workflows", "name": "New"}, {"id": "aa",`, 1)
		f := &fakeBoardClient{board: testBoard, updated: updated}
		_, handler := AddBoardReport(f)
		result := callTool(t, handler, map[string]any{
			"board": "quality", "kind": "time-series", "name": "Duration",
			"params":  map[string]any{"measure": "execution-duration", "aggregate": "avg"},
			"filters": map[string]any{"workflow": []any{"api"}, "environment": "tkcenv_1"},
		})
		require.False(t, result.IsError, getResultText(result))
		assert.Contains(t, getResultText(result), `"reportId":"dd"`)

		u := f.lastUpdate(t)
		assert.Equal(t, "Keep me", *u.Description)
		assert.Nil(t, u.Layout, "the Control Plane places a new report itself")
		require.NotNil(t, u.Content)
		assert.Equal(t, "create", u.Content.Action)
		assert.Equal(t, "report", u.Content.ContentKind)
		p := u.Content.ContentData.Params
		assert.Equal(t, "execution-duration", p["measure"])
		assert.Equal(t, "avg", p["aggregate"])
		assert.Equal(t, "week", p["duration"])
		assert.Equal(t, "bar", p["chartType"])
		filters, err := boards.ParseFilters(p["filter"])
		require.NoError(t, err)
		assert.Equal(t, []string{"api"}, boards.FilterValues(filters, boards.FilterWorkflow))
		assert.Equal(t, []string{"tkcenv_1"}, boards.FilterValues(filters, boards.FilterEnvironment))
	})

	t.Run("rejects unknown params before touching the board", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard}
		_, handler := AddBoardReport(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "kind": "workflows", "name": "W", "params": map[string]any{"measure": "count"}})
		assert.True(t, result.IsError)
		assert.Contains(t, getResultText(result), "unknown params")
		assert.Empty(t, f.updates)
	})

	t.Run("accepts params as a JSON string", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard}
		_, handler := AddBoardReport(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "kind": "pass-fail", "name": "P", "params": `{"measure":"failed-count"}`})
		require.False(t, result.IsError, getResultText(result))
		assert.Equal(t, "failed-count", f.lastUpdate(t).Content.ContentData.Params["measure"])
	})
}

func TestUpdateBoardReport(t *testing.T) {
	t.Run("rejects an unknown report", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard}
		_, handler := UpdateBoardReport(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "reportId": "zz", "name": "x"})
		assert.True(t, result.IsError)
		assert.Contains(t, getResultText(result), `report "zz" not found`)
		assert.Empty(t, f.updates)
	})

	t.Run("merges params and keeps the rest of the report", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard}
		_, handler := UpdateBoardReport(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "reportId": "bb", "params": map[string]any{"measure": "duration"}})
		require.False(t, result.IsError, getResultText(result))
		u := f.lastUpdate(t)
		assert.Equal(t, "Keep me", *u.Description)
		assert.Equal(t, "update", u.Content.Action)
		assert.Equal(t, "bb", u.Content.ContentID)
		d := u.Content.ContentData
		assert.Equal(t, "executions", d.Kind)
		assert.Equal(t, "Exec", d.Name)
		assert.Equal(t, "duration", d.Params["measure"])
		assert.Equal(t, "workflow", d.Params["groupBy"], "unchanged params survive a merge")
		filters, _ := boards.ParseFilters(d.Params["filter"])
		assert.Equal(t, []string{"api"}, boards.FilterValues(filters, boards.FilterWorkflow))
	})

	t.Run("replaceParams starts from defaults", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard}
		_, handler := UpdateBoardReport(f)
		callTool(t, handler, map[string]any{"board": "quality", "reportId": "bb", "replaceParams": true, "params": map[string]any{"measure": "duration"}})
		assert.Equal(t, "status", f.lastUpdate(t).Content.ContentData.Params["groupBy"])
	})

	t.Run("changing the kind drops the old params", func(t *testing.T) {
		f := &fakeBoardClient{board: testBoard}
		_, handler := UpdateBoardReport(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "reportId": "bb", "kind": "workflows"})
		require.False(t, result.IsError, getResultText(result))
		d := f.lastUpdate(t).Content.ContentData
		assert.Equal(t, "workflows", d.Kind)
		assert.NotContains(t, d.Params, "groupBy")
	})
}

func TestRemoveBoardReport(t *testing.T) {
	f := &fakeBoardClient{board: testBoard}
	tool, handler := RemoveBoardReport(f)
	require.NotNil(t, tool.Annotations.DestructiveHint)
	assert.True(t, *tool.Annotations.DestructiveHint)

	result := callTool(t, handler, map[string]any{"board": "quality", "reportId": "cc"})
	require.False(t, result.IsError, getResultText(result))
	u := f.lastUpdate(t)
	assert.Equal(t, "delete", u.Content.Action)
	assert.Equal(t, "cc", u.Content.ContentID)
	assert.Equal(t, "Keep me", *u.Description)
	var layout boards.Layout
	require.NoError(t, json.Unmarshal(u.Layout, &layout))
	assert.Equal(t, [][]string{{"aa", "bb"}}, layout.RowIDs(), "the cell and its now-empty row are dropped")

	result = callTool(t, handler, map[string]any{"board": "quality", "reportId": "zz"})
	assert.True(t, result.IsError)
}

func TestDeleteBoard(t *testing.T) {
	f := &fakeBoardClient{board: testBoard}
	tool, handler := DeleteBoard(f)
	assert.True(t, *tool.Annotations.DestructiveHint)

	result := callTool(t, handler, map[string]any{"board": "quality"})
	require.False(t, result.IsError, getResultText(result))
	assert.Equal(t, "tkcbrd_1", f.deleted)
	assert.Contains(t, getResultText(result), `Deleted board "Quality"`)
	assert.Contains(t, getResultText(result), "3 report(s)")
}

func TestRenderBoard(t *testing.T) {
	newClient := func() *fakeBoardClient {
		return &fakeBoardClient{
			board: testBoard,
			queryResult: map[boards.Endpoint]string{
				boards.EndpointStats:      `{"ratioStats":{"total":90,"values":[["2026-09-01",80],["2026-09-02",100]]},"totalStats":{"total":10,"values":[]},"failedStats":{"total":1,"values":[]}}`,
				boards.EndpointExecutions: `{"count":{"total":5,"values":[["api",5]]},"duration":{"total":0,"values":[]}}`,
			},
			queryErr: map[boards.Endpoint]error{},
		}
	}

	t.Run("renders every report and isolates failures", func(t *testing.T) {
		f := newClient()
		f.queryErr[boards.EndpointExecutions] = fmt.Errorf("API returned status 403: forbidden")
		_, handler := RenderBoard(f)
		result := callTool(t, handler, map[string]any{"board": "quality"})
		require.False(t, result.IsError, getResultText(result))

		var out struct {
			Scope   string `json:"scope"`
			Reports []struct {
				ID    string         `json:"id"`
				Query map[string]any `json:"query"`
				Data  map[string]any `json:"data"`
				Error string         `json:"error"`
			} `json:"reports"`
		}
		require.NoError(t, json.Unmarshal([]byte(getResultText(result)), &out))
		assert.Equal(t, "board", out.Scope)
		require.Len(t, out.Reports, 3)
		assert.Equal(t, []string{"aa", "bb", "cc"}, []string{out.Reports[0].ID, out.Reports[1].ID, out.Reports[2].ID})

		assert.Equal(t, float64(90), out.Reports[0].Data["total"])
		assert.Equal(t, "(all environments)", out.Reports[0].Query["env"])
		assert.Contains(t, out.Reports[1].Error, "Insights may not be enabled")
		assert.Contains(t, out.Reports[2].Error, "no measure", "a time-series report without a measure is not queried")
		assert.Len(t, f.queries, 2)
	})

	t.Run("environment scope asks for the current environment", func(t *testing.T) {
		f := newClient()
		_, handler := RenderBoard(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "reportId": "aa", "scope": "environment"})
		require.False(t, result.IsError, getResultText(result))
		require.Len(t, f.queries, 1)
		assert.True(t, f.queries[0].CurrentEnvironment)
		assert.Contains(t, getResultText(result), "(current environment)")
	})
}

func TestRenderBoard_TimeZone(t *testing.T) {
	f := &fakeBoardClient{
		board:       testBoard,
		queryResult: map[boards.Endpoint]string{boards.EndpointStats: `{"ratioStats":{"total":1,"values":[]},"totalStats":{"total":0,"values":[]},"failedStats":{"total":0,"values":[]}}`},
		queryErr:    map[boards.Endpoint]error{},
	}
	_, handler := RenderBoard(f)

	result := callTool(t, handler, map[string]any{"board": "quality", "reportId": "aa", "timeZone": "Asia/Tokyo"})
	require.False(t, result.IsError, getResultText(result))
	assert.Contains(t, getResultText(result), `"timeZone":"Asia/Tokyo"`)
	require.Len(t, f.queries, 1)
	// A relative range ends at the start of tomorrow in Tokyo, which is 15:00 UTC.
	end := f.queries[0].EndDate.In(time.FixedZone("JST", 9*60*60))
	assert.Equal(t, 0, end.Hour())
	assert.Equal(t, 15, end.UTC().Hour())

	result = callTool(t, handler, map[string]any{"board": "quality", "timeZone": "Nowhere/Land"})
	assert.True(t, result.IsError)
	assert.Contains(t, getResultText(result), "IANA time zone")
}

// boardAt returns the test board as read at updatedAt, with a description.
func boardAt(updatedAt, description string) string {
	b := strings.Replace(testBoard, `"shared": true,`, `"shared": true, "updatedAt": "`+updatedAt+`",`, 1)
	return strings.Replace(b, `"description": "Keep me"`, `"description": "`+description+`"`, 1)
}

func TestBoardWrites_SendTheUpdatedAtTheyRead(t *testing.T) {
	const at = "2026-09-24T10:00:00.123Z"
	tests := []struct {
		name string
		tool func(BoardEditor) (mcp.Tool, server.ToolHandlerFunc)
		args map[string]any
	}{
		{"update_board", UpdateBoard, map[string]any{"board": "quality", "name": "Renamed"}},
		{"add_board_report", AddBoardReport, map[string]any{"board": "quality", "kind": "workflows", "name": "W"}},
		{"update_board_report", UpdateBoardReport, map[string]any{"board": "quality", "reportId": "bb", "name": "Renamed"}},
		{"remove_board_report", RemoveBoardReport, map[string]any{"board": "quality", "reportId": "cc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeBoardClient{board: boardAt(at, "Keep me")}
			_, handler := tt.tool(f)
			result := callTool(t, handler, tt.args)
			require.False(t, result.IsError, getResultText(result))
			assert.Equal(t, at, f.lastUpdate(t).ExpectedUpdatedAt)
		})
	}
}

func TestBoardWrites_RebuildFromTheNewerBoardAfterAConflict(t *testing.T) {
	const first, second = "2026-09-24T10:00:00Z", "2026-09-24T10:00:05Z"
	changed := fmt.Errorf("%w: API returned status 409", ErrBoardChanged)

	t.Run("a concurrent description edit is kept", func(t *testing.T) {
		f := &fakeBoardClient{
			boardReads: []string{boardAt(first, "old"), boardAt(second, "edited meanwhile")},
			updateErrs: []error{changed, nil},
		}
		_, handler := UpdateBoard(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "name": "Renamed"})
		require.False(t, result.IsError, getResultText(result))

		require.Len(t, f.updates, 2)
		retry := f.updates[1]
		assert.Equal(t, second, retry.ExpectedUpdatedAt)
		assert.Equal(t, "edited meanwhile", *retry.Description, "the stale description must not be written back")
		assert.Equal(t, "Renamed", *retry.Name)
	})

	t.Run("a report removal is recomputed from the newer layout", func(t *testing.T) {
		newer := strings.Replace(boardAt(second, "Keep me"),
			`{"id": "r2", "cells": [{"id": "cc"}]}`,
			`{"id": "r2", "cells": [{"id": "cc"}, {"id": "bb"}]}`, 1)
		newer = strings.Replace(newer, `[{"id": "aa"}, {"id": "bb"}]`, `[{"id": "aa"}]`, 1)
		f := &fakeBoardClient{
			boardReads: []string{boardAt(first, "Keep me"), newer},
			updateErrs: []error{changed, nil},
		}
		_, handler := RemoveBoardReport(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "reportId": "cc"})
		require.False(t, result.IsError, getResultText(result))

		var layout boards.Layout
		require.NoError(t, json.Unmarshal(f.lastUpdate(t).Layout, &layout))
		assert.Equal(t, [][]string{{"aa"}, {"bb"}}, layout.RowIDs(), "the move of bb made meanwhile must survive the removal")
	})

	t.Run("a report edit merges into the newer params", func(t *testing.T) {
		newer := strings.Replace(boardAt(second, "Keep me"), `"groupBy": "workflow"`, `"groupBy": "status"`, 1)
		f := &fakeBoardClient{
			boardReads: []string{boardAt(first, "Keep me"), newer},
			updateErrs: []error{changed, nil},
		}
		_, handler := UpdateBoardReport(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "reportId": "bb", "params": map[string]any{"measure": "duration"}})
		require.False(t, result.IsError, getResultText(result))

		params := f.lastUpdate(t).Content.ContentData.Params
		assert.Equal(t, "duration", params["measure"])
		assert.Equal(t, "status", params["groupBy"], "a param changed meanwhile and not named by the caller must be kept")
	})

	t.Run("a report removed meanwhile is reported, not recreated", func(t *testing.T) {
		gone := strings.Replace(boardAt(second, "Keep me"), `{"id": "bb", "kind": "executions"`, `{"id": "zz", "kind": "executions"`, 1)
		f := &fakeBoardClient{
			boardReads: []string{boardAt(first, "Keep me"), gone},
			updateErrs: []error{changed},
		}
		_, handler := UpdateBoardReport(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "reportId": "bb", "name": "x"})
		assert.True(t, result.IsError)
		assert.Contains(t, getResultText(result), `report "bb" not found`)
		assert.Len(t, f.updates, 1)
	})

	t.Run("gives up after a bounded number of attempts", func(t *testing.T) {
		f := &fakeBoardClient{
			board:      boardAt(first, "Keep me"),
			updateErrs: []error{changed, changed, changed, nil},
		}
		_, handler := UpdateBoard(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "name": "Renamed"})
		assert.True(t, result.IsError)
		assert.Contains(t, getResultText(result), "kept changing")
		assert.Len(t, f.updates, boardWriteAttempts)
	})

	t.Run("other errors are not retried", func(t *testing.T) {
		f := &fakeBoardClient{
			board:      boardAt(first, "Keep me"),
			updateErrs: []error{errors.New("API returned status 403: forbidden")},
		}
		_, handler := UpdateBoard(f)
		result := callTool(t, handler, map[string]any{"board": "quality", "name": "Renamed"})
		assert.True(t, result.IsError)
		assert.Len(t, f.updates, 1)
	})
}
