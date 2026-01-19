package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ResourceGroupsLister interface {
	ListResourceGroups(ctx context.Context) (string, error)
}

func ListResourceGroups(client ResourceGroupsLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_resource_groups",
		mcp.WithDescription(ListResourceGroupsDescription),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := client.ListResourceGroups(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}
