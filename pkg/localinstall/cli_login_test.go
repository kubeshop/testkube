package localinstall

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeLogin(t *testing.T, dexStatus int, key string) *httptest.Server {
	names := map[string]bool{}
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
				Name   string              `json:"name"`
				Scopes []map[string]string `json:"scopes"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if names[body.Name] {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			names[body.Name] = true
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

func TestCLIKey_ASecondKeyIsNotRefusedAsADuplicate(t *testing.T) {
	s := fakeLogin(t, http.StatusOK, "tkcapi_abc")
	defer s.Close()

	_, err := cliKey(context.Background(), s.Client(), s.URL, s.URL, "jane@acme.com")
	require.NoError(t, err)
	time.Sleep(2 * time.Millisecond)
	_, err = cliKey(context.Background(), s.Client(), s.URL, s.URL, "jane@acme.com")

	assert.NoError(t, err, "the API refuses a repeated key name")
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

func TestCLIKeyWorks_ARejectedKeyIsReplaced(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tkcapi_good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"id":"` + DemoOrgID + `"}`))
	}))
	defer s.Close()

	assert.True(t, cliKeyWorks(context.Background(), s.Client(), s.URL, "tkcapi_good"))
	assert.False(t, cliKeyWorks(context.Background(), s.Client(), s.URL, "tkcapi_from_wiped_data"))
	assert.False(t, cliKeyWorks(context.Background(), s.Client(), s.URL, ""))
}
