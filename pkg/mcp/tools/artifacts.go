package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ArtifactLister interface {
	ListArtifacts(ctx context.Context, executionId string) (string, error)
}

func ListArtifacts(client ArtifactLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_artifacts",
		mcp.WithDescription("Retrieves all artifacts generated during a workflow execution. Use this tool to discover available outputs, reports, logs, or other files produced by test runs. These artifacts provide valuable context for understanding test results, accessing detailed reports, or examining generated data. The response includes artifact names, sizes, and their current status."),
		mcp.WithString("executionId", mcp.Required(), mcp.Description(ExecutionIdDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionId, err := RequiredParam[string](request, "executionId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := client.ListArtifacts(ctx, executionId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list artifacts: %v", err)), nil
		}

		return CreateToolResultWithDebug(result, client), nil
	}

	return tool, handler
}

const MaxLines = 100

type ArtifactReader interface {
	ReadArtifact(ctx context.Context, executionId, filename string) (string, error)
}

func ReadArtifact(client ArtifactReader) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("read_artifact",
		mcp.WithDescription("Retrieves the content of a specific artifact from a workflow execution. This tool fetches up to 100 lines of text content from the requested file."),
		mcp.WithString("executionId", mcp.Required(), mcp.Description(ExecutionIdDescription)),
		mcp.WithString("fileName", mcp.Required(), mcp.Description(FilenameDescription)),
	)

	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionId, err := RequiredParam[string](request, "executionId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		fileName, err := RequiredParam[string](request, "fileName")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		content, err := client.ReadArtifact(ctx, executionId, fileName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error reading artifact: %v", err)), nil
		}

		if content == "" {
			return mcp.NewToolResultText(fmt.Sprintf("Artifact \"%s\" is empty or not found", fileName)), nil
		}

		// Limit content to max lines
		limitedContent := LimitContentToLines(content, MaxLines)
		return CreateToolResultWithDebug(limitedContent, client), nil
	}

	return tool, handler
}
