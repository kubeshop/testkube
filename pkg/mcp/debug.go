package mcp

import (
	"context"
)

type DebugInfo struct {
	Source string         `json:"source"` // "http", "file", "database", "cache", etc.
	Data   map[string]any `json:"data"`   // Source-specific debug data
}

func NewDebugInfo() *DebugInfo {
	return &DebugInfo{
		Data: make(map[string]any),
	}
}

type contextKey string

const debugInfoKey contextKey = "debug_info"

func WithDebugInfo(ctx context.Context) (context.Context, *DebugInfo) {
	debugInfo := NewDebugInfo()
	newCtx := context.WithValue(ctx, debugInfoKey, debugInfo)
	return newCtx, debugInfo
}

func GetDebugInfo(ctx context.Context) *DebugInfo {
	if debugInfo, ok := ctx.Value(debugInfoKey).(*DebugInfo); ok {
		return debugInfo
	}
	return nil
}
