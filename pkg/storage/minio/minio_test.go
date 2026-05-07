package minio

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetBucketName(t *testing.T) {

	tests := []struct {
		name         string
		parentName   string
		parentType   string
		expectedName string
	}{
		{
			name:         "bucket name is less than 63 chars",
			parentName:   "testName",
			parentType:   "test",
			expectedName: "test-testName",
		},
		{
			name:         "bucket name is 63 chars",
			parentName:   "O7s7A6qyDqtHO6kBDPOjQms0Mgom5P7IQx2W68BAET2Sox00EwMTeJVs1V",
			parentType:   "test",
			expectedName: "test-O7s7A6qyDqtHO6kBDPOjQms0Mgom5P7IQx2W68BAET2Sox00EwMTeJVs1V",
		},
		{
			name: "bucket name is over 63 chars",
			parentName: "O7s7A6qyDqtHO6kBDPOjQms0Mgom5P7IQx2W68BAET2Sox00EwMTeJVs1V" +
				"O7s7A6qyDqtHO6kBDPOjQms0Mgom5P7IQx2W68BAET2Sox00EwMTeJVs1V",
			parentType:   "test",
			expectedName: "test-O7s7A6qyDqtHO6kBDPOjQms0Mgom5P7IQx2W68BAET2Sox0-3877779712",
		},
	}
	var c Client
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			actualName := c.GetValidBucketName(tt.parentType, tt.parentName)
			assert.Equal(t, tt.expectedName, actualName)
			assert.LessOrEqual(t, len(actualName), 63)
		})
	}
}

// TestVirtualHostedStyleURLConstruction verifies that enabling VirtualHostedStyle causes
// the MinIO client to place the bucket name in the request Host header (DNS-style) rather
// than in the URL path (path-style).
func TestVirtualHostedStyleURLConstruction(t *testing.T) {
	t.Parallel()

	const bucket = "my-bucket"

	tests := []struct {
		name                  string
		useVirtualHostedStyle bool
		// When virtual-hosted-style is used, the bucket name appears in the Host header.
		// When path-style is used (default), the bucket name appears in the URL path.
		expectBucketInHost bool
	}{
		{
			name:                  "virtual hosted style enabled - bucket in host",
			useVirtualHostedStyle: true,
			expectBucketInHost:    true,
		},
		{
			name:                  "virtual hosted style disabled - bucket in path",
			useVirtualHostedStyle: false,
			expectBucketInHost:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var requestHost, requestPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestHost = r.Host
				requestPath = r.URL.Path
				// Return minimal error so minio-go doesn't retry or panic.
				w.WriteHeader(http.StatusForbidden)
			}))
			defer srv.Close()

			// Strip "http://" to get just host:port for the minio endpoint.
			serverAddr := strings.TrimPrefix(srv.URL, "http://")

			var opts []Option
			if tt.useVirtualHostedStyle {
				opts = append(opts, VirtualHostedStyle())
			}

			// Use a custom transport that rewrites the target host to our test server
			// while preserving the original Host header so we can assert on it.
			opts = append(opts, withTransport(&rewritingTransport{target: serverAddr}))

			connecter := NewConnecter(
				serverAddr,
				"access-key",
				"secret-key",
				"us-east-1",
				"",
				bucket,
				zap.NewNop().Sugar(),
				opts...,
			)

			client, err := connecter.GetClient()
			require.NoError(t, err)

			// Trigger a request (BucketExists is lightweight and always issues one request).
			_, _ = client.BucketExists(context.Background(), bucket)

			if tt.expectBucketInHost {
				assert.True(t, strings.HasPrefix(requestHost, bucket+"."),
					"expected bucket %q in Host header, got: %q", bucket, requestHost)
				assert.False(t, strings.Contains(requestPath, bucket),
					"expected bucket %q NOT in path for virtual-hosted-style, got path: %q", bucket, requestPath)
			} else {
				assert.False(t, strings.HasPrefix(requestHost, bucket+"."),
					"expected bucket %q NOT in Host header for path-style, got: %q", bucket, requestHost)
				assert.True(t, strings.HasPrefix(requestPath, "/"+bucket),
					"expected bucket %q in path for path-style, got path: %q", bucket, requestPath)
			}
		})
	}
}

// withTransport returns a Connecter Option that replaces the transport used by the minio client.
// This allows tests to intercept or redirect requests.
func withTransport(transport http.RoundTripper) Option {
	return func(o *Connecter) error {
		o.customTransport = transport
		return nil
	}
}

// rewritingTransport rewrites the host:port of each outgoing request to the given target
// while preserving the original Host header (which carries the virtual-hosted-style bucket prefix).
type rewritingTransport struct {
	target string
}

func (t *rewritingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Clone so we don't mutate the original.
	r2 := r.Clone(r.Context())
	// Preserve the original URL.Host as the explicit Host header so the test server
	// receives the bucket-prefixed hostname (e.g. "my-bucket.127.0.0.1:PORT") that
	// minio builds for virtual-hosted-style. Without this, Go's transport falls back
	// to the rewritten URL.Host and the assertion on r.Host in the handler fails.
	if r2.Host == "" {
		r2.Host = r.URL.Host
	}
	// Redirect the actual TCP connection to the test server.
	r2.URL.Host = t.target
	resp, err := http.DefaultTransport.RoundTrip(r2)
	if err != nil {
		// If the server is down, return an empty 403 so the test can check the captured host.
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	}
	return resp, nil
}
