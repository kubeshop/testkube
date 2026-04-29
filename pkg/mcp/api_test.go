package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/mcp/tools"
)

func TestAPIClient_ReadArtifact_UsesArtifactLookupPost(t *testing.T) {
	const (
		orgID       = "org-123"
		envID       = "env-456"
		executionID = "exec-789"
		artifactID  = "results/final-junit.xml"
		content     = "<testsuites/>"
		token       = "test-token"
	)

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/organizations/"+orgID+"/environments/"+envID+"/test-workflow-executions/"+executionID+"/artifacts":
			assert.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var body map[string]string
			err := json.NewDecoder(r.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, artifactID, body["artifactID"])

			w.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(map[string]string{
				"url": serverURL + "/download/artifact",
			})
			assert.NoError(t, err)

		case r.Method == http.MethodGet && r.URL.Path == "/download/artifact":
			_, err := io.WriteString(w, content)
			assert.NoError(t, err)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewAPIClient(&MCPServerConfig{
		ControlPlaneUrl: server.URL,
		AccessToken:     token,
		OrgId:           orgID,
		EnvId:           envID,
	}, server.Client())

	body, err := client.ReadArtifact(context.Background(), executionID, artifactID, tools.ArtifactReadParams{})
	require.NoError(t, err)
	assert.Equal(t, content, body)
}
