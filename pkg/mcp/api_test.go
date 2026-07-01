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

func TestAPIClient_GetExecutionLogs_PropagatesProblemDetail(t *testing.T) {
	const (
		orgID       = "org-1"
		envID       = "env-1"
		executionID = "exec-1"
		token       = "tok"
		detail      = `invalid filter: step "rtff5nk" not found; available steps: init, rwj87r6, rhhrw5b`
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"title":  "Bad Request",
			"status": 400,
			"detail": detail,
		})
	}))
	defer server.Close()

	client := NewAPIClient(&MCPServerConfig{
		ControlPlaneUrl: server.URL,
		AccessToken:     token,
		OrgId:           orgID,
		EnvId:           envID,
	}, server.Client())

	_, err := client.GetExecutionLogs(context.Background(), executionID, tools.ExecutionLogParams{Step: "rtff5nk"})
	require.Error(t, err)
	// The problem+json detail (invalid step + available steps) must reach the caller,
	// not just the bare status code.
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "available steps: init, rwj87r6, rhhrw5b")
}

func TestAPIClient_DebugInfo_URLIncludesQueryParams(t *testing.T) {
	const (
		orgID       = "org-2"
		envID       = "env-2"
		executionID = "exec-2"
		token       = "tok"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "log content")
	}))
	defer server.Close()

	client := NewAPIClient(&MCPServerConfig{
		ControlPlaneUrl: server.URL,
		AccessToken:     token,
		OrgId:           orgID,
		EnvId:           envID,
	}, server.Client())

	ctx, debugInfo := WithDebugInfo(context.Background())
	_, err := client.GetExecutionLogs(ctx, executionID, tools.ExecutionLogParams{Step: "run-tests", Tail: 50})
	require.NoError(t, err)

	gotURL, ok := debugInfo.Data["url"].(string)
	require.True(t, ok, "debug info should record the request URL")
	// The recorded URL must include the query params actually sent, not just the base path.
	assert.Contains(t, gotURL, "step=run-tests")
	assert.Contains(t, gotURL, "tail=50")
}

func TestAPIClient_ListCredentials_HitsEnvScopedFilterAll(t *testing.T) {
	const (
		orgID = "org-1"
		envID = "env-1"
		token = "tok"
		body  = `{"elements":[{"name":"github-access-token","type":"secret","reference":"github-access-token"}]}`
	)

	var gotPath, gotQuery, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("filter")
		gotAuth = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()

	client := NewAPIClient(&MCPServerConfig{
		ControlPlaneUrl: server.URL,
		AccessToken:     token,
		OrgId:           orgID,
		EnvId:           envID,
	}, server.Client())

	result, err := client.ListCredentials(context.Background())
	require.NoError(t, err)
	assert.Equal(t, body, result)
	assert.Equal(t, "/organizations/"+orgID+"/environments/"+envID+"/credentials", gotPath)
	assert.Equal(t, "all", gotQuery)
	assert.Equal(t, "Bearer "+token, gotAuth)
}

func TestExtractErrorDetail(t *testing.T) {
	t.Run("problem json detail", func(t *testing.T) {
		got := extractErrorDetail([]byte(`{"title":"Bad Request","detail":"boom"}`))
		assert.Equal(t, "boom", got)
	})
	t.Run("falls back to title", func(t *testing.T) {
		got := extractErrorDetail([]byte(`{"title":"Bad Request"}`))
		assert.Equal(t, "Bad Request", got)
	})
	t.Run("non-json body returned raw", func(t *testing.T) {
		got := extractErrorDetail([]byte("plain text error"))
		assert.Equal(t, "plain text error", got)
	})
	t.Run("empty body", func(t *testing.T) {
		assert.Equal(t, "", extractErrorDetail(nil))
	})
}
