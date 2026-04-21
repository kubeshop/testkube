package cloudlogin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeJWT builds an unsigned JWT with the given payload claims. Header/signature
// are filler — only the payload base64 is meaningful for the helpers under test.
func makeJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	enc := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	return enc([]byte(`{"alg":"none"}`)) + "." + enc(payload) + ".sig"
}

func TestJwtExpired(t *testing.T) {
	now := time.Now()
	tests := map[string]struct {
		token string
		skew  time.Duration
		want  bool
	}{
		"valid token with future exp":    {token: makeJWT(t, map[string]any{"exp": now.Add(10 * time.Minute).Unix()}), skew: time.Minute, want: false},
		"token expired past skew":        {token: makeJWT(t, map[string]any{"exp": now.Add(-time.Minute).Unix()}), skew: time.Minute, want: true},
		"token expiring within skew":     {token: makeJWT(t, map[string]any{"exp": now.Add(30 * time.Second).Unix()}), skew: time.Minute, want: true},
		"missing exp claim":              {token: makeJWT(t, map[string]any{"sub": "abc"}), skew: time.Minute, want: true},
		"malformed token (not 3 parts)":  {token: "abc.def", skew: time.Minute, want: true},
		"malformed payload (bad base64)": {token: "abc.!!!.sig", skew: time.Minute, want: true},
		"empty token":                    {token: "", skew: time.Minute, want: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, jwtExpired(tc.token, tc.skew))
		})
	}
}

func TestEmailFromIDToken(t *testing.T) {
	tests := map[string]struct {
		token string
		want  string
	}{
		"email claim present":          {token: makeJWT(t, map[string]any{"email": "user@example.com"}), want: "user@example.com"},
		"missing email claim":          {token: makeJWT(t, map[string]any{"sub": "abc"}), want: ""},
		"malformed token":              {token: "not-a-jwt", want: ""},
		"empty token":                  {token: "", want: ""},
		"email alongside other claims": {token: makeJWT(t, map[string]any{"email": "a@b.c", "exp": 1, "sub": "x"}), want: "a@b.c"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, EmailFromIDToken(tc.token))
		})
	}
}

func TestRequestEmailLink_SendsExpectedRequest(t *testing.T) {
	var gotPath, gotContentType, gotEmail, gotState string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotEmail = r.PostFormValue("email")
		gotState = r.PostFormValue("state")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	redirect := getRedirectAddress(8090)
	err := requestEmailLink(context.Background(), srv.URL+"/", "user@example.com", redirect)
	require.NoError(t, err)
	assert.Equal(t, "/auth/link", gotPath)
	assert.Equal(t, "application/x-www-form-urlencoded", gotContentType)
	assert.Equal(t, "user@example.com", gotEmail)
	assert.Equal(t, "redirect="+redirect, gotState)
}

func TestRequestEmailLink_ResponseHandling(t *testing.T) {
	tests := map[string]struct {
		status      int
		body        string
		wantErr     bool
		wantErrSubs []string
	}{
		"200 returns nil":          {status: http.StatusOK},
		"400 surfaces status+body": {status: http.StatusBadRequest, body: "bad email", wantErr: true, wantErrSubs: []string{"status 400", "bad email"}},
		"500 surfaces status+body": {status: http.StatusInternalServerError, body: "boom", wantErr: true, wantErrSubs: []string{"status 500", "boom"}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			err := requestEmailLink(context.Background(), srv.URL+"/", "x@y.z", getRedirectAddress(8090))
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, sub := range tc.wantErrSubs {
				assert.Contains(t, err.Error(), sub)
			}
		})
	}
}

func TestExchangeOOBCodeForTokens(t *testing.T) {
	t.Run("decodes idToken and refreshToken on success", func(t *testing.T) {
		var gotEmail, gotOOBCode string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/auth/login/link", r.URL.Path)
			gotEmail = r.URL.Query().Get("email")
			gotOOBCode = r.URL.Query().Get("oobCode")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"idToken":"id-xyz","refreshToken":"refresh-abc"}`))
		}))
		defer srv.Close()

		tokens, err := exchangeOOBCodeForTokens(context.Background(), srv.URL+"/", "user@example.com", "oob-123")
		require.NoError(t, err)
		assert.Equal(t, "user@example.com", gotEmail)
		assert.Equal(t, "oob-123", gotOOBCode)
		assert.Equal(t, "id-xyz", tokens.IDToken)
		assert.Equal(t, "refresh-abc", tokens.RefreshToken)
	})

	t.Run("non-200 response surfaces error with status and body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("invalid oobCode"))
		}))
		defer srv.Close()

		_, err := exchangeOOBCodeForTokens(context.Background(), srv.URL+"/", "a@b.c", "bad")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 401")
		assert.Contains(t, err.Error(), "invalid oobCode")
	})
}

func TestRefreshEmailLinkToken(t *testing.T) {
	t.Run("skips network when current token still valid", func(t *testing.T) {
		hits := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		validToken := makeJWT(t, map[string]any{"exp": time.Now().Add(10 * time.Minute).Unix()})
		newID, newRefresh, err := RefreshEmailLinkToken(context.Background(), srv.URL+"/", validToken, "refresh-keep")
		require.NoError(t, err)
		assert.Equal(t, validToken, newID)
		assert.Equal(t, "refresh-keep", newRefresh)
		assert.Equal(t, 0, hits, "server should not have been called")
	})

	t.Run("refreshes when current token is expired", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/auth/callback", r.URL.Path)
			assert.Equal(t, "emailLink", r.URL.Query().Get("method"))
			body, _ := io.ReadAll(r.Body)
			assert.Contains(t, string(body), `"refreshToken":"old-refresh"`)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"idToken":"new-id","refreshToken":"new-refresh","accessToken":"new-id"}`))
		}))
		defer srv.Close()

		expired := makeJWT(t, map[string]any{"exp": time.Now().Add(-time.Minute).Unix()})
		newID, newRefresh, err := RefreshEmailLinkToken(context.Background(), srv.URL+"/", expired, "old-refresh")
		require.NoError(t, err)
		assert.Equal(t, "new-id", newID)
		assert.Equal(t, "new-refresh", newRefresh)
	})

	t.Run("refreshes when current idToken is empty", func(t *testing.T) {
		hits := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"idToken":"fresh","refreshToken":"r"}`))
		}))
		defer srv.Close()

		newID, newRefresh, err := RefreshEmailLinkToken(context.Background(), srv.URL+"/", "", "r0")
		require.NoError(t, err)
		assert.Equal(t, 1, hits)
		assert.Equal(t, "fresh", newID)
		assert.Equal(t, "r", newRefresh)
	})

	t.Run("surfaces non-200 from refresh endpoint", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("refresh rejected"))
		}))
		defer srv.Close()

		expired := makeJWT(t, map[string]any{"exp": time.Now().Add(-time.Minute).Unix()})
		_, _, err := RefreshEmailLinkToken(context.Background(), srv.URL+"/", expired, "refresh")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 401")
	})
}

func TestCloudLoginEmailLink_RequestFailurePropagates(t *testing.T) {
	// Control plane refuses the link request; ensure the function does not
	// leak the loopback listener when the upstream POST errors.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("disabled"))
	}))
	defer srv.Close()

	port := freeTestPort(t)
	_, err := CloudLoginEmailLink(context.Background(), srv.URL+"/", "u@e.com", port)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 403")

	// If the listener leaked, a follow-up bind on the same port would fail.
	require.NoError(t, checkPortAvailable(port), "port should be released after upstream error")
}

// freeTestPort finds a port currently free on 127.0.0.1 for use by a test.
func freeTestPort(t *testing.T) int {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}
