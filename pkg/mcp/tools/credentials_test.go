package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCredentialsLister struct {
	called    bool
	returnRaw string
	returnErr error
}

func (m *mockCredentialsLister) ListCredentials(_ context.Context) (string, error) {
	m.called = true
	return m.returnRaw, m.returnErr
}

func callListCredentials(t *testing.T, mock *mockCredentialsLister) (*mcp.CallToolResult, error) {
	t.Helper()
	_, handler := ListCredentials(mock)
	return handler(context.Background(), mcp.CallToolRequest{})
}

func TestListCredentials_HappyPath_FormatsReferences(t *testing.T) {
	m := &mockCredentialsLister{returnRaw: `{"elements":[
		{"name":"github-access-token","type":"secret","reference":"github-access-token"}
	]}`}

	result, err := callListCredentials(t, m)
	require.NoError(t, err)
	require.True(t, m.called, "tool must invoke the client with no required arguments")
	require.NotNil(t, result)
	require.False(t, result.IsError)

	text, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "github-access-token")
	assert.Contains(t, text.Text, `credential(\"github-access-token\")`)
}

func TestListCredentials_ClientError_ReturnsToolError(t *testing.T) {
	m := &mockCredentialsLister{returnErr: fmt.Errorf("API returned status 403: forbidden")}

	result, err := callListCredentials(t, m)
	require.NoError(t, err) // handler never returns a Go error
	require.NotNil(t, result)
	require.True(t, result.IsError)

	text, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "forbidden")
}
