package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPHTTPStartupErrors(t *testing.T) {
	t.Setenv("TK_MCP_ENV_MODE", "true")
	t.Setenv("TK_ACCESS_TOKEN", "outbound-token")
	t.Setenv("TK_ORG_ID", "org")
	t.Setenv("TK_ENV_ID", "env")
	t.Setenv("DOCKER_CONTAINER", "true")
	for _, token := range []string{"", "invalid token", "outbound-token"} {
		t.Setenv("TK_MCP_SHTTP_AUTH_TOKEN", token)
		cmd := NewMcpServeCmd()
		output := &bytes.Buffer{}
		cmd.SetOut(output)
		cmd.SetErr(output)
		cmd.SetArgs([]string{"--transport=shttp"})
		require.ErrorContains(t, cmd.Execute(), "TK_MCP_SHTTP_AUTH_TOKEN")
		host, err := cmd.Flags().GetString("shttp-host")
		require.NoError(t, err)
		assert.Equal(t, "localhost", host)
		if token != "" {
			assert.NotContains(t, output.String(), token)
		}
	}
	// Both credential modes use this shared startup function. A reused token proves
	// the environment value is wired before any API client or optional config load.
	for _, source := range []string{"cli-env", "cli-direct"} {
		t.Setenv("TK_MCP_SHTTP_AUTH_TOKEN", "outbound-token")
		err := startMCPServer(false, "outbound-token", "org", "env", "", "", false, "shttp", "", 0, false, "", "", source)
		require.ErrorContains(t, err, "must differ")
	}
}
