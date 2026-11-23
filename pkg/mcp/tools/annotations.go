package tools

import "github.com/mark3labs/mcp-go/mcp"

// Every tool sets all four hints: a hint left out falls back to the spec's riskiest value.

// readOnlyTool marks a tool that only reads Testkube data.
func readOnlyTool() mcp.ToolOption { return toolHints(true, false, true, false) }

// additiveTool marks a tool that only creates something new.
func additiveTool() mcp.ToolOption { return toolHints(false, false, false, false) }

// destructiveTool marks a tool that overwrites or removes existing data.
func destructiveTool() mcp.ToolOption { return toolHints(false, true, true, false) }

func toolHints(readOnly, destructive, idempotent, openWorld bool) mcp.ToolOption {
	return func(t *mcp.Tool) {
		t.Annotations.ReadOnlyHint = &readOnly
		t.Annotations.DestructiveHint = &destructive
		t.Annotations.IdempotentHint = &idempotent
		t.Annotations.OpenWorldHint = &openWorld
	}
}
