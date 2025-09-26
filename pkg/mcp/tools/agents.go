package tools

import (
	"context"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ListAgentsParams struct {
	Type           string `json:"type,omitempty"`           // Filter by agent type (e.g., "runner")
	Capability     string `json:"capability,omitempty"`     // Filter by capability (e.g., "runner")
	PageSize       int    `json:"pageSize,omitempty"`       // Number of items per page (default: 20)
	Page           int    `json:"page,omitempty"`           // Page number (default: 0)
	IncludeDeleted bool   `json:"includeDeleted,omitempty"` // Include deleted agents (default: false)
}

type AgentsLister interface {
	ListAgents(ctx context.Context, params ListAgentsParams) (string, error)
}

func ListAgents(client AgentsLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_agents",
		mcp.WithDescription(ListAgentsDescription),
		mcp.WithString("type", mcp.Description("Filter by agent type (e.g., 'runner')")),
		mcp.WithString("capability", mcp.Description("Filter by capability (e.g., 'runner')")),
		mcp.WithString("pageSize", mcp.Description("Number of items per page (default: 20)")),
		mcp.WithString("page", mcp.Description("Page number (default: 0)")),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := ListAgentsParams{
			Type:       request.GetString("type", ""),
			Capability: request.GetString("capability", ""),
		}

		// Parse pageSize
		if pageSizeStr := request.GetString("pageSize", "20"); pageSizeStr != "" {
			if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
				params.PageSize = pageSize
			} else {
				params.PageSize = 20
			}
		} else {
			params.PageSize = 20
		}

		// Parse page
		if pageStr := request.GetString("page", "0"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil && page >= 0 {
				params.Page = page
			} else {
				params.Page = 0
			}
		} else {
			params.Page = 0
		}

		// Call the API
		result, err := client.ListAgents(ctx, params)
		if err != nil {
			return mcp.NewToolResultError("Error listing agents: " + err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}
