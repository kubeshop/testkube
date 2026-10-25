package mcpcontext

import (
	"context"
)

// DebugInfo collects what a tool call did, for the debug output of the
// call. It is not safe for concurrent use: a tool that makes several client
// calls in parallel gives each its own DebugInfo (see WithDebugInfo) and
// merges them once they are done.
type DebugInfo struct {
	Source string         `json:"source"` // "http", "file", "database", "cache", etc.
	Data   map[string]any `json:"data"`   // Source-specific debug data
}

func NewDebugInfo() *DebugInfo {
	return &DebugInfo{
		Data: make(map[string]any),
	}
}

const debugInfoKey contextKey = "debug_info"

// WithDebugInfo returns a context carrying a new DebugInfo, and that DebugInfo.
func WithDebugInfo(ctx context.Context) (context.Context, *DebugInfo) {
	debugInfo := NewDebugInfo()
	newCtx := context.WithValue(ctx, debugInfoKey, debugInfo)
	return newCtx, debugInfo
}

// GetDebugInfo returns the DebugInfo in ctx, or nil when debugging is off.
func GetDebugInfo(ctx context.Context) *DebugInfo {
	if debugInfo, ok := ctx.Value(debugInfoKey).(*DebugInfo); ok {
		return debugInfo
	}
	return nil
}
