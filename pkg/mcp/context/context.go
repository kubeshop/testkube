package mcpcontext

import (
	"context"
)

type contextKey string

// Context keys for per-request org/env IDs.
// In CLI mode these are static for the server lifetime, but in cloud mode
// they change per HTTP request. Tools that need org/env should read these
// from context as a fallback when their initialization-time values are empty.
const (
	orgIDKey contextKey = "mcp_org_id"
	envIDKey contextKey = "mcp_env_id"
)

// WithOrgEnv returns a context carrying the given organization and environment IDs.
func WithOrgEnv(ctx context.Context, orgID, envID string) context.Context {
	ctx = context.WithValue(ctx, orgIDKey, orgID)
	ctx = context.WithValue(ctx, envIDKey, envID)
	return ctx
}

// GetOrgID returns the organization ID stored in ctx, or "" if absent.
func GetOrgID(ctx context.Context) string {
	if id, ok := ctx.Value(orgIDKey).(string); ok {
		return id
	}
	return ""
}

// GetEnvID returns the environment ID stored in ctx, or "" if absent.
func GetEnvID(ctx context.Context) string {
	if id, ok := ctx.Value(envIDKey).(string); ok {
		return id
	}
	return ""
}
