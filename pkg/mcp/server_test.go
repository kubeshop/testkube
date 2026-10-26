package mcp

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shttpTestConfig() MCPServerConfig {
	return MCPServerConfig{Version: "test", AccessToken: "outbound-token", OrgId: "org", EnvId: "env", SkipEndpointChecks: true,
		SHTTPConfig: SHTTPConfig{AuthToken: "inbound-token", Host: "127.0.0.1"}}
}

type unreadBody struct{ reads int }

func (b *unreadBody) Read([]byte) (int, error) { b.reads++; return 0, io.EOF }
func (*unreadBody) Close() error               { return nil }

func TestSHTTPAuthentication(t *testing.T) {
	for _, method := range []string{"POST", "GET", "DELETE", "HEAD", "OPTIONS"} {
		for _, headers := range [][]string{nil, {""}, {"Bearer "}, {"Basic inbound-token"}, {"Bearer wrong"}, {"Bearer outbound-token"}, {"Bearer  inbound-token"}, {"Bearer\tinbound-token"}, {"Bearer inbound-token "}, {"Bearer inbound-token, Bearer inbound-token"}, {"Bearer inbound-token", "Bearer inbound-token"}} {
			t.Run(method+"/"+strings.Join(headers, ","), func(t *testing.T) {
				called := false
				handler := authenticateSHTTP("inbound-token", http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
				req := httptest.NewRequestWithContext(t.Context(), method, "/mcp?access_token=inbound-token", nil)
				req.Header["Authorization"] = headers
				req.Header.Set("Mcp-Session-Id", "previous-session")
				req.AddCookie(&http.Cookie{Name: "token", Value: "inbound-token"})
				body := &unreadBody{}
				req.Body = body
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, req)
				assert.Equal(t, http.StatusUnauthorized, response.Code)
				assert.Equal(t, "Bearer", response.Header().Get("WWW-Authenticate"))
				assert.False(t, called)
				assert.Zero(t, body.reads)
			})
		}
	}
	for _, scheme := range []string{"Bearer", "bearer", "bEaReR"} {
		response := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), "POST", "/mcp", nil)
		req.Header.Set("Authorization", scheme+" inbound-token")
		authenticateSHTTP("inbound-token", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(response, req)
		assert.Equal(t, http.StatusNoContent, response.Code)
	}
}

func TestSHTTPConfiguration(t *testing.T) {
	for _, token := range []string{"", " ", "bad token", "bad\ntoken", "bad:token", "=bad", "bad=middle", "outbound-token"} {
		cfg := shttpTestConfig()
		cfg.SHTTPConfig.AuthToken = token
		// Transport is deliberately unset: this exported entry point must still fail closed.
		require.Error(t, ServeSHTTPMCP(cfg, nil))
	}
	for _, token := range []string{"valid._~+/-==", "inbound-token"} {
		cfg := shttpTestConfig()
		cfg.SHTTPConfig.AuthToken = token
		cfg.SHTTPConfig.Host = ""
		_, srv, err := newSHTTPServer(cfg, nil)
		require.NoError(t, err)
		assert.Equal(t, "localhost:0", srv.Addr)
	}
	cfg := shttpTestConfig()
	cfg.SHTTPConfig.Host = "0.0.0.0"
	_, srv, err := newSHTTPServer(cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0:0", srv.Addr)
	cfg.SHTTPConfig.AuthToken = ""
	cfg.Transport = TransportStdio
	_, err = NewMCPServer(cfg, nil)
	require.NoError(t, err)
	t.Setenv("TK_MCP_SHTTP_AUTH_TOKEN", "from-environment")
	assert.Equal(t, "from-environment", LoadConfigFromEnv().SHTTPConfig.AuthToken)
}

func TestSHTTPWorkflowCredentialBoundary(t *testing.T) {
	var calls atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/organizations/org/environments/env/agent/test-workflows/example/executions", r.URL.Path)
		assert.Equal(t, "Bearer outbound-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"execution-id","name":"example-1","number":1}`)
	}))
	defer api.Close()
	cfg := shttpTestConfig()
	cfg.ControlPlaneUrl = api.URL
	_, srv, err := newSHTTPServer(cfg, nil)
	require.NoError(t, err)
	endpoint := httptest.NewServer(srv.Handler)
	defer endpoint.Close()
	client := endpoint.Client()
	client.Timeout = 3 * time.Second
	session := ""
	messages := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"run_workflow","arguments":{"workflowName":"example"}}}`,
	}
	for _, message := range messages {
		for _, authenticated := range []bool{false, true} {
			before := calls.Load()
			req, err := http.NewRequestWithContext(t.Context(), "POST", endpoint.URL+"/mcp", strings.NewReader(message))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			req.Header.Set("Mcp-Session-Id", session)
			if authenticated {
				req.Header.Set("Authorization", "Bearer inbound-token")
			}
			resp, err := client.Do(req)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			if !authenticated {
				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				assert.Equal(t, before, calls.Load())
				continue
			}
			require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
			var result map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(body, &result))
			assert.NotContains(t, result, "error")
			if strings.Contains(message, "initialize") {
				session = resp.Header.Get("Mcp-Session-Id")
				require.NotEmpty(t, session)
			}
			if strings.Contains(message, "tools/list") {
				assert.Contains(t, string(body), "run_workflow")
			}
			if strings.Contains(message, "tools/call") {
				assert.Contains(t, string(body), "execution-id")
				assert.NotContains(t, string(body), `"isError":true`)
			}
		}
	}
	assert.EqualValues(t, 1, calls.Load())
	for _, method := range []string{"GET", "DELETE"} {
		req := httptest.NewRequestWithContext(t.Context(), method, "/mcp", nil)
		req.Header.Set("Mcp-Session-Id", session)
		response := httptest.NewRecorder()
		srv.Handler.ServeHTTP(response, req)
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	}
}

func TestSHTTPLifecycle(t *testing.T) {
	for _, tlsEnabled := range []bool{false, true} {
		t.Run(strconv.FormatBool(tlsEnabled), func(t *testing.T) {
			cfg := shttpTestConfig()
			// Reserve a free loopback port; the production Start method owns its listener.
			reservation, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
			require.NoError(t, err)
			cfg.SHTTPConfig.Port = reservation.Addr().(*net.TCPAddr).Port
			require.NoError(t, reservation.Close())
			scheme := "http"
			// Keep the shutdown check focused on the active stream, without an idle client connection pool.
			client := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
			if tlsEnabled {
				certificateServer := httptest.NewTLSServer(http.NotFoundHandler())
				defer certificateServer.Close()
				cert := certificateServer.TLS.Certificates[0]
				key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
				require.NoError(t, err)
				dir := t.TempDir()
				cfg.SHTTPConfig.EnableTLS = true
				cfg.SHTTPConfig.CertFile = filepath.Join(dir, "cert.pem")
				cfg.SHTTPConfig.KeyFile = filepath.Join(dir, "key.pem")
				require.NoError(t, os.WriteFile(cfg.SHTTPConfig.CertFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0600))
				require.NoError(t, os.WriteFile(cfg.SHTTPConfig.KeyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), 0600))
				client = certificateServer.Client()
				client.Transport.(*http.Transport).DisableKeepAlives = true
				client.Timeout = 3 * time.Second
				scheme = "https"
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- serveSHTTP(ctx, cfg, nil) }()
			address := net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.SHTTPConfig.Port))
			url := scheme + "://" + address + "/mcp"
			require.Eventually(t, func() bool {
				req, err := http.NewRequestWithContext(t.Context(), "GET", url, nil)
				if err != nil {
					return false
				}
				resp, err := client.Do(req)
				if err != nil {
					return false
				}
				defer resp.Body.Close()
				_, _ = io.Copy(io.Discard, resp.Body)
				return resp.StatusCode == http.StatusUnauthorized
			}, 5*time.Second, 10*time.Millisecond)
			req, err := http.NewRequestWithContext(t.Context(), "GET", url, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer inbound-token")
			req.Header.Set("Accept", "text/event-stream")
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)
			streamClosed := make(chan error, 1)
			go func() { _, err := io.Copy(io.Discard, resp.Body); streamClosed <- err }()
			cancel()
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(6 * time.Second):
				t.Fatal("shutdown did not finish")
			}
			select {
			case err := <-streamClosed:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("stream remains open")
			}
			listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", address)
			require.NoError(t, err)
			require.NoError(t, listener.Close())
		})
	}
	cfg := shttpTestConfig()
	cfg.SHTTPConfig.EnableTLS = true
	cfg.SHTTPConfig.CertFile = "missing-cert"
	cfg.SHTTPConfig.KeyFile = "missing-key"
	require.Error(t, serveSHTTP(t.Context(), cfg, nil))
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	cfg = shttpTestConfig()
	cfg.SHTTPConfig.Port = listener.Addr().(*net.TCPAddr).Port
	require.Error(t, serveSHTTP(t.Context(), cfg, nil))
}
