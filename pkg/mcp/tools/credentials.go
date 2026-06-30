package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kubeshop/testkube/pkg/mcp/formatters"
)

type CredentialsLister interface {
	ListCredentials(ctx context.Context) (string, error)
}

func ListCredentials(client CredentialsLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_credentials",
		mcp.WithDescription(ListCredentialsDescription),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := client.ListCredentials(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		formatted, err := formatters.FormatListCredentials(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format credentials: %v", err)), nil
		}

		return mcp.NewToolResultText(formatted), nil
	}

	return tool, handler
}
