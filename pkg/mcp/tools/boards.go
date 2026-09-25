package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kubeshop/testkube/pkg/mcp/boards"
	mcpcontext "github.com/kubeshop/testkube/pkg/mcp/context"
	"github.com/kubeshop/testkube/pkg/mcp/formatters"
)

// Board tools manage Insights boards: saved, organization-wide dashboards made
// of reports (charts) over execution data. Boards are organization-scoped, not
// environment-scoped; a report narrows itself to environments through its own
// filter. The Control Plane only serves boards to signed-in users, so these
// tools fail for API tokens (see ErrBoardsRequireUser).
//
// Report params are an opaque object to the Control Plane. The tools validate
// and normalize them with pkg/mcp/boards so the dashboard can render and edit
// what they write.

// ErrBoardsRequireUser is returned by a client when the session authenticates
// with an API token, which the Control Plane refuses for every board endpoint.
var ErrBoardsRequireUser = errors.New("insights boards require a signed-in user session; API tokens are not supported. " +
	"Sign in with `testkube login` and restart the MCP server, or connect to the hosted MCP endpoint with your user account. " +
	"In Docker or environment-variable mode, TK_ACCESS_TOKEN must be a user access token, not an API token (tkcapi_...)")

// ErrBoardChanged is returned by a client when a conditional board update is
// refused because the board no longer has the version the update carries in
// ExpectedVersion.
var ErrBoardChanged = errors.New("the board changed since it was read")

const renderConcurrency = 4

// boardWriteAttempts bounds how many times a board write is rebuilt from a
// fresh read after losing a race with a concurrent edit.
const boardWriteAttempts = 3

// ListBoardsParams filters the board list.
type ListBoardsParams struct {
	Name         string
	Private      bool
	Shared       bool
	UserFavorite bool
	OrgFavorite  bool
	Page         int
	PageSize     int
}

// CreateBoardParams describes a new board.
type CreateBoardParams struct {
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	Description string `json:"description,omitempty"`
	IsPrivate   bool   `json:"isPrivate"`
}

// BoardContentPatch adds, replaces or removes one report of a board.
type BoardContentPatch struct {
	Action      string              `json:"action"` // create | update | delete
	ContentKind string              `json:"content_kind"`
	ContentID   string              `json:"content_id,omitempty"`
	ContentData *boards.ReportDraft `json:"content_data,omitempty"`
}

// UpdateBoardRequest is the body of a board update. Nil fields are left
// unchanged. The board tools always send Description too: current Control
// Planes keep a description an update omits, but older ones clear it.
//
// Resending a value read earlier would overwrite a concurrent edit of it, so
// every board write also sends ExpectedVersion, the version of the board it
// read. The Control Plane then refuses the write with 409 if the board has
// changed since, and the tools read it again and rebuild the write. A Control
// Plane that predates the field ignores it.
type UpdateBoardRequest struct {
	Name            *string            `json:"name,omitempty"`
	Description     *string            `json:"description,omitempty"`
	Slug            *string            `json:"slug,omitempty"`
	IsPrivate       *bool              `json:"isPrivate,omitempty"`
	Layout          json.RawMessage    `json:"layout,omitempty"`
	Content         *BoardContentPatch `json:"content,omitempty"`
	ExpectedVersion *int64             `json:"expectedVersion,omitempty"`
}

// BoardLister lists the boards visible to the user.
type BoardLister interface {
	ListBoards(ctx context.Context, params ListBoardsParams) (string, error)
}

// BoardGetter returns a board by ID or slug.
type BoardGetter interface {
	GetBoard(ctx context.Context, board string) (string, error)
}

// BoardSlugChecker reports whether a board slug is still free.
type BoardSlugChecker interface {
	CheckBoardSlug(ctx context.Context, slug string) (bool, error)
}

// BoardCreator creates a board.
type BoardCreator interface {
	CreateBoard(ctx context.Context, params CreateBoardParams) (string, error)
}

// BoardUpdater updates a board's details, layout or reports.
type BoardUpdater interface {
	UpdateBoard(ctx context.Context, board string, request UpdateBoardRequest) (string, error)
}

// BoardDeleter deletes a board.
type BoardDeleter interface {
	DeleteBoard(ctx context.Context, board string) error
}

// BoardInsightQuerier runs the insight query that renders a board report and
// returns the raw response.
type BoardInsightQuerier interface {
	QueryBoardInsights(ctx context.Context, query boards.InsightQuery) (string, error)
}

// BoardEditor reads a board and then writes it.
type BoardEditor interface {
	BoardGetter
	BoardUpdater
}

// BoardCreatorWithSlugCheck creates a board after checking its slug.
type BoardCreatorWithSlugCheck interface {
	BoardCreator
	BoardSlugChecker
}

// BoardRemover reads a board and then deletes it.
type BoardRemover interface {
	BoardGetter
	BoardDeleter
}

// BoardRenderer reads a board and runs its reports' queries.
type BoardRenderer interface {
	BoardGetter
	BoardInsightQuerier
}

// ListBoards creates a tool for listing Insights boards.
func ListBoards(client BoardLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_boards",
		mcp.WithDescription(ListBoardsDescription),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("name", mcp.Description("Filter boards whose name contains this text.")),
		mcp.WithString("visibility", mcp.Description("'all' (default), 'private' (only your private boards) or 'shared' (only boards shared with the organization)."), mcp.Enum("all", "private", "shared")),
		mcp.WithString("favorite", mcp.Description("'user' for boards you pinned, 'org' for boards pinned for the organization."), mcp.Enum("user", "org")),
		mcp.WithNumber("page", mcp.Description(PageDescription)),
		mcp.WithNumber("pageSize", mcp.Description("Number of boards per page (default: 20, max: 100)")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := OptionalParam[string](request, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		visibility, err := OptionalParam[string](request, "visibility")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		favorite, err := OptionalParam[string](request, "favorite")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		page, err := OptionalIntParam(request, "page")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		pageSize, err := OptionalIntParamWithDefault(request, "pageSize", 20)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if page < 0 {
			page = 0
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = min(max(pageSize, 1), 100)
		}

		params := ListBoardsParams{Name: name, Page: page, PageSize: pageSize}
		switch visibility {
		case "", "all":
		case "private":
			params.Private = true
		case "shared":
			params.Shared = true
		default:
			return mcp.NewToolResultError("visibility must be one of all, private, shared"), nil
		}
		switch favorite {
		case "":
		case "user":
			params.UserFavorite = true
		case "org":
			params.OrgFavorite = true
		default:
			return mcp.NewToolResultError("favorite must be one of user, org"), nil
		}

		result, err := client.ListBoards(ctx, params)
		if err != nil {
			return boardError("list boards", err), nil
		}
		formatted, err := formatters.FormatBoardList(result, page, pageSize)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format boards: %v", err)), nil
		}
		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

// GetBoard creates a tool for viewing a board and its reports.
func GetBoard(client BoardGetter) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("get_board",
		mcp.WithDescription(GetBoardDescription),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("board", mcp.Required(), mcp.Description(BoardIdDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		board, err := RequiredParam[string](request, "board")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result, err := client.GetBoard(ctx, board)
		if err != nil {
			return boardError("get board", err), nil
		}
		formatted, err := formatters.FormatBoard(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format board: %v", err)), nil
		}
		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

// CreateBoard creates a tool for creating an empty board.
func CreateBoard(client BoardCreatorWithSlugCheck) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("create_board",
		mcp.WithDescription(CreateBoardDescription),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the board.")),
		mcp.WithString("description", mcp.Description("Description of the board.")),
		mcp.WithString("slug", mcp.Description("URL-friendly identifier. Generated from the name when omitted; must be unused in the organization.")),
		mcp.WithBoolean("private", mcp.Description("Create the board private to you instead of shared with the organization (default: false, shared).")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := RequiredParam[string](request, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		description, err := OptionalParam[string](request, "description")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		slug, err := OptionalParam[string](request, "slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		private, err := OptionalParam[bool](request, "private")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// The Control Plane does not check a slug it is given on create.
		if slug = strings.TrimSpace(slug); slug != "" {
			available, err := client.CheckBoardSlug(ctx, slug)
			if err != nil {
				return boardError("check board slug", err), nil
			}
			if !available {
				return mcp.NewToolResultError(fmt.Sprintf("slug %q is already used by another board; choose another or omit it to generate one", slug)), nil
			}
		}

		result, err := client.CreateBoard(ctx, CreateBoardParams{Name: name, Slug: slug, Description: description, IsPrivate: private})
		if err != nil {
			return boardError("create board", err), nil
		}
		formatted, err := formatters.FormatBoard(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format board: %v", err)), nil
		}
		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

// UpdateBoard creates a tool for changing a board's details or layout.
func UpdateBoard(client BoardEditor) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("update_board",
		mcp.WithDescription(UpdateBoardDescription),
		mcp.WithString("board", mcp.Required(), mcp.Description(BoardIdDescription)),
		mcp.WithString("name", mcp.Description("New name.")),
		mcp.WithString("description", mcp.Description("New description. Pass an empty string to clear it; omit to keep it.")),
		mcp.WithString("slug", mcp.Description("New URL-friendly identifier; must be unused in the organization.")),
		mcp.WithBoolean("private", mcp.Description("true makes the board private to its creator, false shares it with the organization. Only the creator or an organization admin can change this.")),
		mcp.WithObject("layout", mcp.Description(BoardLayoutDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		boardRef, err := RequiredParam[string](request, "board")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		// The arguments: every field the caller did not name keeps its value.
		var change UpdateBoardRequest
		var layout *boards.Layout
		if name, ok, err := OptionalParamOK[string](request, "name"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		} else if ok && strings.TrimSpace(name) != "" {
			change.Name = &name
		}
		if description, ok, err := OptionalParamOK[string](request, "description"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		} else if ok {
			change.Description = &description
		}
		if slug, ok, err := OptionalParamOK[string](request, "slug"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		} else if ok && strings.TrimSpace(slug) != "" {
			slug = strings.TrimSpace(slug)
			change.Slug = &slug
		}
		if private, ok, err := OptionalParamOK[bool](request, "private"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		} else if ok {
			change.IsPrivate = &private
		}
		if raw, ok := request.GetArguments()["layout"]; ok && raw != nil {
			parsed, err := parseLayoutParam(raw)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			layout = &parsed
		}
		if change.Name == nil && change.Description == nil && change.Slug == nil && change.IsPrivate == nil && layout == nil {
			return mcp.NewToolResultError("nothing to update: pass at least one of name, description, slug, private, layout"), nil
		}

		result, _, errResult := writeBoard(ctx, client, boardRef, "update board", func(board *boards.Board) (UpdateBoardRequest, *mcp.CallToolResult) {
			req := change
			if req.Description == nil {
				req.Description = &board.Description
			}
			if layout != nil {
				// Checked against the board as it is now, so a report added
				// since the caller read the board is not silently unplaced.
				if err := layout.Validate(board.ReportIDs()); err != nil {
					return req, mcp.NewToolResultError(err.Error())
				}
				req.Layout, _ = json.Marshal(layout)
			}
			return req, nil
		})
		if errResult != nil {
			return errResult, nil
		}
		formatted, err := formatters.FormatBoard(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format board: %v", err)), nil
		}
		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

// AddBoardReport creates a tool for adding a report to a board.
func AddBoardReport(client BoardEditor) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("add_board_report",
		mcp.WithDescription(AddBoardReportDescription),
		mcp.WithString("board", mcp.Required(), mcp.Description(BoardIdDescription)),
		mcp.WithString("kind", mcp.Required(), mcp.Description(BoardReportKindDescription), mcp.Enum(boards.Kinds...)),
		mcp.WithString("name", mcp.Required(), mcp.Description("Title of the report.")),
		mcp.WithString("description", mcp.Description("Description of the report.")),
		mcp.WithObject("params", mcp.Description(BoardReportParamsDescription)),
		mcp.WithObject("filters", mcp.Description(BoardReportFiltersDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		boardRef, err := RequiredParam[string](request, "board")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		kind, err := RequiredParam[string](request, "kind")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := RequiredParam[string](request, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		description, err := OptionalParam[string](request, "description")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		params, filters, err := reportInput(request, kind)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if params == nil {
			params = map[string]any{}
		}
		if err := boards.ApplyFilters(params, filters); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		normalized, err := boards.NormalizeReport(kind, params)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, board, errResult := writeBoard(ctx, client, boardRef, "add report", func(board *boards.Board) (UpdateBoardRequest, *mcp.CallToolResult) {
			return UpdateBoardRequest{
				Description: &board.Description,
				Content: &BoardContentPatch{
					Action:      "create",
					ContentKind: "report",
					ContentData: &boards.ReportDraft{Kind: kind, Name: name, Description: description, Params: normalized},
				},
			}, nil
		})
		if errResult != nil {
			return errResult, nil
		}
		before := board.ReportIDs()
		return boardChangeResult(result, func(updated *boards.Board) string {
			for _, id := range updated.ReportIDs() {
				if !slices.Contains(before, id) {
					return id
				}
			}
			return ""
		})
	}

	return tool, handler
}

// UpdateBoardReport creates a tool for changing a report on a board.
func UpdateBoardReport(client BoardEditor) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("update_board_report",
		mcp.WithDescription(UpdateBoardReportDescription),
		mcp.WithString("board", mcp.Required(), mcp.Description(BoardIdDescription)),
		mcp.WithString("reportId", mcp.Required(), mcp.Description("ID of the report to change (from get_board).")),
		mcp.WithString("name", mcp.Description("New title.")),
		mcp.WithString("description", mcp.Description("New description. Pass an empty string to clear it; omit to keep it.")),
		mcp.WithString("kind", mcp.Description("New report kind. Changing it starts from that kind's defaults instead of the old params."), mcp.Enum(boards.Kinds...)),
		mcp.WithObject("params", mcp.Description(BoardReportParamsDescription+" Merged into the current params unless replaceParams is true; a null value removes a param.")),
		mcp.WithObject("filters", mcp.Description(BoardReportFiltersDescription+" Each key given replaces that key's current filters; an empty list removes them.")),
		mcp.WithBoolean("replaceParams", mcp.Description("Replace the params entirely instead of merging (default: false).")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		boardRef, err := RequiredParam[string](request, "board")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		reportID, err := RequiredParam[string](request, "reportId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		replace, err := OptionalParam[bool](request, "replaceParams")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		newKind, err := OptionalParam[string](request, "kind")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		newName, err := OptionalParam[string](request, "name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		newDescription, hasDescription, err := OptionalParamOK[string](request, "description")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, _, errResult := writeBoard(ctx, client, boardRef, "update report", func(board *boards.Board) (UpdateBoardRequest, *mcp.CallToolResult) {
			// The Control Plane silently ignores an update of an unknown report.
			existing, ok := board.FindReport(reportID)
			if !ok {
				return UpdateBoardRequest{}, reportNotFound(board, reportID)
			}

			// The merge starts from the report as it is now, so a concurrent
			// edit of params the caller did not name is kept.
			draft := boards.ReportDraft{Kind: existing.Kind, Name: existing.Name, Description: existing.Description}
			replaceParams := replace
			if newKind != "" && newKind != existing.Kind {
				draft.Kind = newKind
				replaceParams = true
			}
			if strings.TrimSpace(newName) != "" {
				draft.Name = newName
			}
			if hasDescription {
				draft.Description = newDescription
			}

			patch, filters, err := reportInput(request, draft.Kind)
			if err != nil {
				return UpdateBoardRequest{}, mcp.NewToolResultError(err.Error())
			}
			base := existing.Params
			if replaceParams {
				base = nil
			}
			params := boards.MergeParams(base, patch)
			if err := boards.ApplyFilters(params, filters); err != nil {
				return UpdateBoardRequest{}, mcp.NewToolResultError(err.Error())
			}
			if draft.Params, err = boards.NormalizeReport(draft.Kind, params); err != nil {
				return UpdateBoardRequest{}, mcp.NewToolResultError(err.Error())
			}

			return UpdateBoardRequest{
				Description: &board.Description,
				Content: &BoardContentPatch{
					Action:      "update",
					ContentKind: "report",
					ContentID:   reportID,
					ContentData: &draft,
				},
			}, nil
		})
		if errResult != nil {
			return errResult, nil
		}
		return boardChangeResult(result, func(*boards.Board) string { return reportID })
	}

	return tool, handler
}

// RemoveBoardReport creates a tool for removing a report from a board.
func RemoveBoardReport(client BoardEditor) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("remove_board_report",
		mcp.WithDescription(RemoveBoardReportDescription),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("board", mcp.Required(), mcp.Description(BoardIdDescription)),
		mcp.WithString("reportId", mcp.Required(), mcp.Description("ID of the report to remove (from get_board).")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		boardRef, err := RequiredParam[string](request, "board")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		reportID, err := RequiredParam[string](request, "reportId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, _, errResult := writeBoard(ctx, client, boardRef, "remove report", func(board *boards.Board) (UpdateBoardRequest, *mcp.CallToolResult) {
			if _, ok := board.FindReport(reportID); !ok {
				return UpdateBoardRequest{}, reportNotFound(board, reportID)
			}
			// The Control Plane leaves the layout alone when a report is
			// deleted, so send the layout without the report's cell alongside
			// the delete - derived from the board as it is now.
			layout, err := boards.ParseLayout(board.Layout)
			if err != nil {
				return UpdateBoardRequest{}, mcp.NewToolResultError(err.Error())
			}
			layoutJSON, _ := json.Marshal(layout.Without(reportID))
			return UpdateBoardRequest{
				Description: &board.Description,
				Layout:      layoutJSON,
				Content:     &BoardContentPatch{Action: "delete", ContentKind: "report", ContentID: reportID},
			}, nil
		})
		if errResult != nil {
			return errResult, nil
		}
		formatted, err := formatters.FormatBoard(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format board: %v", err)), nil
		}
		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

// DeleteBoard creates a tool for deleting a board.
func DeleteBoard(client BoardRemover) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("delete_board",
		mcp.WithDescription(DeleteBoardDescription),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("board", mcp.Required(), mcp.Description(BoardIdDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		boardRef, err := RequiredParam[string](request, "board")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		board, errResult := fetchBoard(ctx, client, boardRef)
		if errResult != nil {
			return errResult, nil
		}
		if err := client.DeleteBoard(ctx, board.ID); err != nil {
			return boardError("delete board", err), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Deleted board %q (id: %s, slug: %s) and its %d report(s).", board.Name, board.ID, board.Slug, len(board.Content.Reports))), nil
	}

	return tool, handler
}

// renderedReport is one report's entry in the render_board result.
type renderedReport struct {
	ID    string         `json:"id"`
	Name  string         `json:"name,omitempty"`
	Kind  string         `json:"kind"`
	Query map[string]any `json:"query,omitempty"`
	Data  any            `json:"data,omitempty"`
	Error string         `json:"error,omitempty"`
}

// RenderBoard creates a tool that runs a board's reports and returns their data.
func RenderBoard(client BoardRenderer) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("render_board",
		mcp.WithDescription(RenderBoardDescription),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("board", mcp.Required(), mcp.Description(BoardIdDescription)),
		mcp.WithString("reportId", mcp.Description("Render only this report (from get_board). Renders every report when omitted.")),
		mcp.WithString("scope", mcp.Description(BoardRenderScopeDescription), mcp.Enum("board", "environment")),
		mcp.WithString("timeZone", mcp.Description(BoardRenderTimeZoneDescription)),
		mcp.WithNumber("maxSamples", mcp.Description(InsightMaxSamplesDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		boardRef, err := RequiredParam[string](request, "board")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		reportID, err := OptionalParam[string](request, "reportId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		scope, err := OptionalParam[string](request, "scope")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if scope == "" {
			scope = "board"
		}
		if scope != "board" && scope != "environment" {
			return mcp.NewToolResultError("scope must be one of board, environment"), nil
		}
		maxSamples, err := OptionalIntParam(request, "maxSamples")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		maxSamples = min(max(maxSamples, 0), 500)
		timeZone, err := OptionalParam[string](request, "timeZone")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		loc, err := boards.ParseTimeZone(timeZone)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		// One instant for every report, so they all cover the same range.
		opts := boards.QueryOptions{Now: time.Now(), CurrentEnvironment: scope == "environment", Location: loc}

		board, errResult := fetchBoard(ctx, client, boardRef)
		if errResult != nil {
			return errResult, nil
		}
		reports, unplaced, err := board.OrderedReports()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if reportID != "" {
			// A report asked for by ID is rendered even if the layout leaves
			// it out.
			r, ok := board.FindReport(reportID)
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("report %q not found on board %q (reports: %s)", reportID, board.Name, strings.Join(board.ReportIDs(), ", "))), nil
			}
			reports = []boards.Report{*r}
		} else {
			// Render what the dashboard shows. Reports the layout leaves out
			// are listed under unplaced, not queried.
			reports = reports[:len(reports)-len(unplaced)]
		}

		results := make([]renderedReport, len(reports))
		// The clients record each call in the context's DebugInfo, which is
		// not safe for concurrent use. When debugging is on, every report
		// query gets its own, merged into the call's once all are done.
		debug := mcpcontext.GetDebugInfo(ctx)
		reportDebug := make([]*mcpcontext.DebugInfo, len(reports))
		sem := make(chan struct{}, renderConcurrency)
		var wg sync.WaitGroup
		for i, r := range reports {
			reportCtx := ctx
			if debug != nil {
				reportCtx, reportDebug[i] = mcpcontext.WithDebugInfo(ctx)
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				results[i] = renderReport(reportCtx, client, r, opts, maxSamples)
			}()
		}
		wg.Wait()
		if debug != nil {
			perReport := make(map[string]*mcpcontext.DebugInfo, len(reports))
			for i, r := range reports {
				perReport[r.ID] = reportDebug[i]
			}
			debug.Data["reports"] = perReport
		}

		out := struct {
			Board    string           `json:"board"`
			Name     string           `json:"name"`
			Scope    string           `json:"scope"`
			TimeZone string           `json:"timeZone"`
			Reports  []renderedReport `json:"reports"`
			Unplaced []string         `json:"unplaced,omitempty"`
		}{Board: board.ID, Name: board.Name, Scope: scope, TimeZone: loc.String(), Reports: results}
		if reportID == "" {
			out.Unplaced = unplaced
		}
		formatted, err := formatters.FormatJSON(out)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format board data: %v", err)), nil
		}
		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}

func renderReport(ctx context.Context, client BoardInsightQuerier, r boards.Report, opts boards.QueryOptions, maxSamples int) renderedReport {
	out := renderedReport{ID: r.ID, Name: r.Name, Kind: r.Kind}
	q, err := boards.BuildQuery(r.Kind, r.Params, opts)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.Query = map[string]any{"endpoint": string(q.Endpoint)}
	for k, v := range q.QueryParams() {
		out.Query[k] = v
	}
	if q.CurrentEnvironment {
		out.Query["env"] = "(current environment)"
	} else if q.Env == "" {
		out.Query["env"] = "(all environments)"
	}

	raw, err := client.QueryBoardInsights(ctx, q)
	if err != nil {
		out.Error = insightErrorMessage(err)
		return out
	}
	measure, _ := r.Params["measure"].(string)
	data, err := formatters.ReportData(r.Kind, measure, raw, maxSamples)
	if err != nil {
		out.Error = fmt.Sprintf("failed to read report data: %v", err)
		return out
	}
	out.Data = data
	return out
}

// fetchBoard loads a board for a tool that changes it. Every write reads the
// board first: to resend its description, to check a report exists, and to
// address it by ID even when the caller passed a slug.
func fetchBoard(ctx context.Context, client BoardGetter, ref string) (*boards.Board, *mcp.CallToolResult) {
	raw, err := client.GetBoard(ctx, ref)
	if err != nil {
		return nil, boardError("get board", err)
	}
	board, err := boards.ParseBoard(raw)
	if err != nil {
		return nil, mcp.NewToolResultError(err.Error())
	}
	return board, nil
}

// writeBoard reads the board, builds an update from it, and sends it
// conditioned on the version it read. build must derive everything it takes
// from the board it is given. When a concurrent edit wins the race, the board
// is read again and the update rebuilt from it, so the caller's change is
// reapplied on top of the newer board instead of overwriting it. It returns
// the updated board and the board the update was built from.
func writeBoard(ctx context.Context, client BoardEditor, ref, action string,
	build func(*boards.Board) (UpdateBoardRequest, *mcp.CallToolResult)) (string, *boards.Board, *mcp.CallToolResult) {
	for attempt := 1; ; attempt++ {
		board, errResult := fetchBoard(ctx, client, ref)
		if errResult != nil {
			return "", nil, errResult
		}
		req, errResult := build(board)
		if errResult != nil {
			return "", nil, errResult
		}
		// A Control Plane that predates versions returns none; then the write
		// is unconditional, as before.
		req.ExpectedVersion = board.Version

		result, err := client.UpdateBoard(ctx, board.ID, req)
		if err == nil {
			return result, board, nil
		}
		if !errors.Is(err, ErrBoardChanged) || attempt == boardWriteAttempts {
			return "", nil, boardError(action, err)
		}
		// Read the same board again by the ID the first read resolved, not by
		// what the caller passed: the concurrent change may have been to the
		// slug, which could now be missing or belong to another board.
		ref = board.ID
	}
}

func reportNotFound(board *boards.Board, reportID string) *mcp.CallToolResult {
	return mcp.NewToolResultError(fmt.Sprintf("report %q not found on board %q (reports: %s)", reportID, board.Name, strings.Join(board.ReportIDs(), ", ")))
}

// boardChangeResult formats a board returned by an update, together with the
// report the update was about.
func boardChangeResult(raw string, reportOf func(*boards.Board) string) (*mcp.CallToolResult, error) {
	board, err := boards.ParseBoard(raw)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	view, err := formatters.BoardView(board)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	out := struct {
		ReportID string                    `json:"reportId,omitempty"`
		Board    formatters.FormattedBoard `json:"board"`
	}{ReportID: reportOf(board), Board: view}
	formatted, err := formatters.FormatJSON(out)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format board: %v", err)), nil
	}
	return mcp.NewToolResultText(formatted), nil
}

// reportInput reads the params and filters arguments of a report tool. Params
// the kind does not understand are rejected.
func reportInput(request mcp.CallToolRequest, kind string) (map[string]any, map[string][]string, error) {
	if !boards.IsKind(kind) {
		return nil, nil, fmt.Errorf("unknown report kind %q (allowed: %s)", kind, strings.Join(boards.Kinds, ", "))
	}
	var params map[string]any
	switch v := request.GetArguments()["params"].(type) {
	case nil:
	case map[string]any:
		params = v
	case string:
		if strings.TrimSpace(v) != "" {
			if err := json.Unmarshal([]byte(v), &params); err != nil {
				return nil, nil, fmt.Errorf("params must be a JSON object: %w", err)
			}
		}
	default:
		return nil, nil, fmt.Errorf("params must be an object, got %T", v)
	}
	if err := boards.CheckParamKeys(kind, params); err != nil {
		return nil, nil, err
	}

	var filters map[string][]string
	switch v := request.GetArguments()["filters"].(type) {
	case nil:
	case map[string]any:
		filters = make(map[string][]string, len(v))
		for key, value := range v {
			switch values := value.(type) {
			case nil:
				filters[key] = nil
			case string:
				filters[key] = []string{values}
			case []any:
				for _, item := range values {
					s, ok := item.(string)
					if !ok {
						return nil, nil, fmt.Errorf("filters.%s must be a list of strings", key)
					}
					filters[key] = append(filters[key], s)
				}
				if filters[key] == nil {
					filters[key] = []string{}
				}
			default:
				return nil, nil, fmt.Errorf("filters.%s must be a list of strings", key)
			}
		}
	default:
		return nil, nil, fmt.Errorf("filters must be an object, got %T", v)
	}
	return params, filters, nil
}

func parseLayoutParam(raw any) (boards.Layout, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return boards.Layout{}, fmt.Errorf("layout is malformed: %w", err)
	}
	if s, ok := raw.(string); ok {
		data = []byte(s)
	}
	var layout boards.Layout
	if err := json.Unmarshal(data, &layout); err != nil {
		return boards.Layout{}, fmt.Errorf("layout must be {\"version\": 1, \"rows\": [{\"cells\": [{\"id\": \"<reportId>\"}]}]}: %w", err)
	}
	return layout, nil
}

// boardError turns a client error into a tool error, replacing the Control
// Plane's refusal of API tokens with an actionable message.
func boardError(action string, err error) *mcp.CallToolResult {
	msg := err.Error()
	switch {
	case errors.Is(err, ErrBoardsRequireUser), strings.Contains(msg, "API tokens are not supported"):
		return mcp.NewToolResultError(ErrBoardsRequireUser.Error())
	case errors.Is(err, ErrBoardChanged):
		return mcp.NewToolResultError(fmt.Sprintf("Failed to %s: the board kept changing while the change was applied (%d attempts), so nothing was written. Someone may be editing it; try again.", action, boardWriteAttempts))
	case strings.Contains(msg, "status 404"):
		return mcp.NewToolResultError(fmt.Sprintf("Failed to %s: board not found (it may be private to another user). Use list_boards to find it. (%v)", action, err))
	case strings.Contains(msg, "status 500"):
		// Older Control Planes answer an API-token caller with a bare 500.
		return mcp.NewToolResultError(fmt.Sprintf("Failed to %s: %v. If this session uses an API token, note that boards require a signed-in user session (`testkube login`).", action, err))
	}
	return mcp.NewToolResultError(fmt.Sprintf("Failed to %s: %v", action, err))
}

func insightErrorMessage(err error) string {
	if strings.Contains(err.Error(), "status 403") {
		return fmt.Sprintf("access denied - Insights may not be enabled for this organization (%v)", err)
	}
	return err.Error()
}
