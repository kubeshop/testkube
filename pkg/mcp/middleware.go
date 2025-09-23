package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/telemetry"
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

// TelemetryMiddleware creates a middleware that collects telemetry for MCP tool execution
func TelemetryMiddleware(telemetryEnabled bool) server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			startTime := time.Now()

			// Execute the tool
			result, err := next(ctx, request)

			// Send telemetry event
			if telemetryEnabled {
				duration := time.Since(startTime)
				hasError := err != nil

				// For now, use a generic tool name since we don't have direct access to the tool name
				// This can be enhanced later with a more sophisticated approach
				toolName := request.Params.Name

				// Send telemetry asynchronously to avoid blocking tool execution
				go func() {
					telemetry.SendMCPToolEvent(toolName, duration, hasError, common.Version)
				}()
			}

			return result, err
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
