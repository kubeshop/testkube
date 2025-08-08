package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DebugMiddleware creates a middleware that automatically collects debug information
// when debug mode is enabled
func DebugMiddleware(enabled bool) server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if !enabled {
				return next(ctx, request)
			}

			debugCtx, debugInfo := WithDebugInfo(ctx)

			result, err := next(debugCtx, request)
			if err != nil {
				return result, err
			}

			if debugInfo == nil || debugInfo.Source == "" {
				return result, nil
			}

			return enhanceResultWithDebug(result, debugInfo), nil
		}
	}
}

// enhanceResultWithDebug adds debug information to the tool result
func enhanceResultWithDebug(result *mcp.CallToolResult, debugInfo *DebugInfo) *mcp.CallToolResult {
	if result == nil || debugInfo == nil {
		return result
	}

	debugJSON, err := json.MarshalIndent(debugInfo, "", "  ")
	if err != nil {
		return result
	}

	debugContent := mcp.NewTextContent("Debug Information:\n" + string(debugJSON))
	result.Content = append(result.Content, debugContent)

	return result
}
