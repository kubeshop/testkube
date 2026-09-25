package mcp

import (
	"context"

	mcpcontext "github.com/kubeshop/testkube/pkg/mcp/context"
)

// DebugInfo lives in mcpcontext so the tools, which this package imports,
// can use it too. These keep the existing names working.
type DebugInfo = mcpcontext.DebugInfo

func NewDebugInfo() *DebugInfo {
	return mcpcontext.NewDebugInfo()
}

func WithDebugInfo(ctx context.Context) (context.Context, *DebugInfo) {
	return mcpcontext.WithDebugInfo(ctx)
}

func GetDebugInfo(ctx context.Context) *DebugInfo {
	return mcpcontext.GetDebugInfo(ctx)
}
