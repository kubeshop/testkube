package localinstall

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeLogin(t *testing.T, dexStatus int, key string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			require.NoError(t, r.ParseForm())
			if dexStatus != http.StatusOK || r.Form.Get("username") != "jane@acme.com" || r.Form.Get("password") != AdminPassword {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"id_token":"id-123"}`))
		case "/organizations/" + DemoOrgID + "/tokens":
			if r.Header.Get("Authorization") != "Bearer id-123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			var body struct {
				Scopes []map[string]string `json:"scopes"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, DemoEnvID, body.Scopes[1]["identifier"], "the key can run the demo environment's workflows")
			_ = json.NewEncoder(w).Encode(map[string]string{"token": key})
		}
	}))
}

func TestCLIKey_LogsInAsTheDashboardUser(t *testing.T) {
	s := fakeLogin(t, http.StatusOK, "tkcapi_abc")
	defer s.Close()

	key, err := cliKey(context.Background(), s.Client(), s.URL, s.URL, "jane@acme.com")

	require.NoError(t, err)
	assert.Equal(t, "tkcapi_abc", key)
}

func TestCLIKey_NeverReturnsAnEmptyKey(t *testing.T) {
	tests := map[string]*httptest.Server{
		"login refused":    fakeLogin(t, http.StatusUnauthorized, "tkcapi_abc"),
		"no key in answer": fakeLogin(t, http.StatusOK, ""),
	}
	for name, s := range tests {
		t.Run(name, func(t *testing.T) {
			defer s.Close()

			key, err := cliKey(context.Background(), s.Client(), s.URL, s.URL, "jane@acme.com")

			assert.ErrorIs(t, err, ErrCLILogin)
			assert.Empty(t, key)
		})
	}
}
