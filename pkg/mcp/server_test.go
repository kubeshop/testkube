package mcp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestMCPServerTools builds an MCP server with the given config and returns
// the set of registered tool names. SkipEndpointChecks avoids the network-bound
// backwards-compat probe so the test stays offline.
func newTestMCPServerTools(t *testing.T, includeCredentialTools bool) map[string]struct{} {
	t.Helper()
	cfg := MCPServerConfig{
		Version:                "test",
		ControlPlaneUrl:        "http://localhost",
		AccessToken:            "test-token",
		OrgId:                  "org-1",
		EnvId:                  "env-1",
		SkipEndpointChecks:     true,
		IncludeCredentialTools: includeCredentialTools,
	}
	client := NewAPIClient(&cfg, http.DefaultClient)
	srv, err := NewMCPServer(cfg, client)
	require.NoError(t, err)

	names := make(map[string]struct{})
	for name := range srv.ListTools() {
		names[name] = struct{}{}
	}
	return names
}

func TestNewMCPServer_CredentialToolGating(t *testing.T) {
	t.Run("list_credentials is registered when the flag is set (hosted surfaces)", func(t *testing.T) {
		tools := newTestMCPServerTools(t, true)
		_, ok := tools["list_credentials"]
		assert.True(t, ok, "expected list_credentials to be registered when IncludeCredentialTools=true")
	})

	t.Run("list_credentials is omitted when the flag is unset (local CLI)", func(t *testing.T) {
		tools := newTestMCPServerTools(t, false)
		_, ok := tools["list_credentials"]
		assert.False(t, ok, "expected list_credentials to be omitted when IncludeCredentialTools=false")

		// Sanity: gating list_credentials must not drop other tools.
		_, hasLabels := tools["list_labels"]
		assert.True(t, hasLabels, "other tools must still be registered regardless of the credential flag")
	})
}
