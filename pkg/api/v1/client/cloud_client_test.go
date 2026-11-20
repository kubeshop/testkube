package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordPaths serves every request with an empty JSON list, except a download-URL
// request, which points at the server's own /signed/file, and records each path.
func recordPaths(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var (
		mu    sync.Mutex
		paths []string
	)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		switch r.URL.Path {
		case "/signed/file":
			_, _ = w.Write([]byte("content"))
		case "/organizations/o/environments/e/test-workflow-executions/ex1/artifacts/report.xml":
			_ = json.NewEncoder(w).Encode(ArtifactURL{URL: server.URL + "/signed/file"})
		default:
			_, _ = w.Write([]byte("[]"))
		}
	}))
	t.Cleanup(server.Close)
	return server, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), paths...)
	}
}

// TestCloudClient_StatesAPIVersion guards that --client=cloud states the API version
// itself, so the Control Plane forwards /v1/<resource> to the agent and never /v1/v1.
func TestCloudClient_StatesAPIVersion(t *testing.T) {
	server, paths := recordPaths(t)
	c := NewCloudAPIClient(server.Client(), server.Client(), server.URL, CloudAgentPathPrefix("o", "e"))

	_, err := c.ListTestWorkflows("")
	require.NoError(t, err)

	assert.Equal(t, []string{"/organizations/o/environments/e/agent/v1/test-workflows"}, paths())
}

// TestCloudClient_GetFileAsksControlPlaneForDownloadURL guards that an artifact
// download asks the Control Plane's own endpoint, outside the versioned agent
// proxy, for the signed URL, then downloads from it.
func TestCloudClient_GetFileAsksControlPlaneForDownloadURL(t *testing.T) {
	server, paths := recordPaths(t)
	c := NewCloudAPIClient(server.Client(), server.Client(), server.URL, CloudAgentPathPrefix("o", "e"))

	_, err := c.DownloadTestWorkflowArtifact("ex1", "report.xml", t.TempDir())
	require.NoError(t, err)

	assert.Equal(t, []string{
		"/organizations/o/environments/e/test-workflow-executions/ex1/artifacts/report.xml",
		"/signed/file",
	}, paths())
}
