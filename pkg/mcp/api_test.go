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

	"github.com/kubeshop/testkube/pkg/mcp/formatters"
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

// TestAPIClient_ListInsightExecutions_ArrayBodyFeedsFormatter guards the CLI path
// end to end: the control-plane endpoint returns a bare JSON array (pagination in
// the Link header, not the body), the APIClient returns that body verbatim, and
// FormatInsightExecutions must parse the array shape without error.
func TestAPIClient_ListInsightExecutions_ArrayBodyFeedsFormatter(t *testing.T) {
	const (
		orgID = "org-3"
		envID = "env-3"
		token = "tok"
	)

	// Real wire shape: ArrStart … ArrEnd plus a Link header for pagination.
	const arrayBody = `[{"id":"e1","name":"n1","parent":"wf","environment":"env-3","status":"passed","duration":1200,"runAt":"2026-07-01T00:00:00Z"}]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/organizations/"+orgID+"/insights/series/executions", r.URL.Path)
		assert.Equal(t, envID, r.URL.Query().Get("env"))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `</next>; rel="next"`)
		_, _ = io.WriteString(w, arrayBody)
	}))
	defer server.Close()

	client := NewAPIClient(&MCPServerConfig{
		ControlPlaneUrl: server.URL,
		AccessToken:     token,
		OrgId:           orgID,
		EnvId:           envID,
	}, server.Client())

	raw, err := client.ListInsightExecutions(context.Background(), tools.InsightExecutionsParams{Measure: "http_req_duration_p95_ms"})
	require.NoError(t, err)
	assert.JSONEq(t, arrayBody, raw)

	// The formatter must accept this array body — this is the call that previously
	// failed with "cannot unmarshal array into Go value of type ...".
	formatted, err := formatters.FormatInsightExecutions(raw)
	require.NoError(t, err)
	assert.Contains(t, formatted, `"id":"e1"`)
	assert.Contains(t, formatted, `"workflow":"wf"`)
	assert.Contains(t, formatted, `"durationMs":1200`)
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
