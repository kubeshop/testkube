package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newReadinessServer(t *testing.T) HTTPServer {
	t.Helper()
	return NewServer(Config{Port: 0})
}

func get(t *testing.T, s HTTPServer, path string) (int, []byte) {
	t.Helper()
	resp, err := s.Mux.Test(httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, body
}

func TestReadyEndpoint_ReadyWithNoChecks(t *testing.T) {
	status, _ := get(t, newReadinessServer(t), "/ready")
	assert.Equal(t, http.StatusOK, status)
}

func TestReadyEndpoint_ReadyWhenEveryCheckPasses(t *testing.T) {
	s := newReadinessServer(t)
	s.AddReadinessCheck("a", func() error { return nil })
	s.AddReadinessCheck("b", func() error { return nil })

	status, _ := get(t, s, "/ready")

	assert.Equal(t, http.StatusOK, status)
}

// A failing check has to name itself, so an operator can tell which subsystem
// stalled without reading the logs.
func TestReadyEndpoint_ReportsTheFailingCheck(t *testing.T) {
	s := newReadinessServer(t)
	s.AddReadinessCheck("healthy-one", func() error { return nil })
	s.AddReadinessCheck("runner-execution-updates", func() error {
		return errors.New("no successful execution update poll for 6h0m0s")
	})

	status, body := get(t, s, "/ready")

	require.Equal(t, http.StatusServiceUnavailable, status)
	var payload struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, "not ready", payload.Status)
	assert.Contains(t, payload.Checks, "runner-execution-updates")
	assert.NotContains(t, payload.Checks, "healthy-one")
}

// The registry is shared across copies, because NewServer returns HTTPServer by
// value and main registers checks on a copy.
func TestReadyEndpoint_RegistrySurvivesValueCopy(t *testing.T) {
	s := newReadinessServer(t)
	copied := s
	copied.AddReadinessCheck("registered-on-a-copy", func() error { return errors.New("down") })

	status, _ := get(t, s, "/ready")

	assert.Equal(t, http.StatusServiceUnavailable, status,
		"a check registered on a copy has to be visible to the original")
}

// /health stays unconditional so existing probes and tooling keep working; only
// readiness reflects the poll loop.
func TestHealthEndpoint_StaysUnconditional(t *testing.T) {
	s := newReadinessServer(t)
	s.AddReadinessCheck("broken", func() error { return errors.New("down") })

	status, body := get(t, s, "/health")

	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), "OK")
}
