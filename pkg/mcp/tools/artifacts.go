package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ArtifactLister interface {
	ListArtifacts(ctx context.Context, executionId string) (string, error)
}

func ListArtifacts(client ArtifactLister) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("list_artifacts",
		mcp.WithDescription(ListArtifactsDescription),
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

		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}

type ArtifactReader interface {
	ReadArtifact(ctx context.Context, executionId, filename string, params ArtifactReadParams) (string, error)
}

func ReadArtifact(client ArtifactReader) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	tool = mcp.NewTool("read_artifact",
		mcp.WithDescription(ReadArtifactDescription),
		mcp.WithString("executionId", mcp.Required(), mcp.Description(ExecutionIdDescription)),
		mcp.WithString("fileName", mcp.Required(), mcp.Description(FilenameDescription)),
		mcp.WithString("startLine",
			mcp.Description("1-based line number to start reading from. Use with endLine for a range."),
		),
		mcp.WithString("endLine",
			mcp.Description("1-based line number to stop reading at (inclusive). Use with startLine for a range."),
		),
		mcp.WithString("grep",
			mcp.Description("Case-insensitive substring filter. Returns matching lines with 3 lines of context."),
		),
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

		var params ArtifactReadParams
		params.Grep = request.GetString("grep", "")

		if s := request.GetString("startLine", ""); s != "" {
			v, err := strconv.Atoi(s)
			if err != nil || v <= 0 {
				return mcp.NewToolResultError(fmt.Sprintf("startLine must be a positive integer, got %q", s)), nil
			}
			params.StartLine = v
		}
		if s := request.GetString("endLine", ""); s != "" {
			v, err := strconv.Atoi(s)
			if err != nil || v <= 0 {
				return mcp.NewToolResultError(fmt.Sprintf("endLine must be a positive integer, got %q", s)), nil
			}
			params.EndLine = v
		}

		content, err := client.ReadArtifact(ctx, executionId, fileName, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error reading artifact: %v", err)), nil
		}

		if content == "" {
			return mcp.NewToolResultText(fmt.Sprintf("Artifact \"%s\" is empty or not found", fileName)), nil
		}

		result := ProcessArtifact([]byte(content), fileName, params)
		return mcp.NewToolResultText(result), nil
	}

	return tool, handler
}
