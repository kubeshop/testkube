package client

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "github.com/kubeshop/testkube/pkg/api/v1/client"
)

func TestNewImportClient(t *testing.T) {
	c := NewImportClient("https://api.testkube.io", "my-token", "org-1", "env-1")

	assert.Equal(t, "https://api.testkube.io", c.BaseUrl)
	assert.Equal(t, "/organizations/org-1/environments/env-1/agent/import", c.Path)
	assert.Equal(t, "my-token", c.Token)
	assert.NotNil(t, c.Client)
}

func createTestArchive(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "testkube-export-*.tar.gz")
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	data := []byte(`{"id":"exec-1"}`)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "test.json", Size: int64(len(data)), Mode: 0o644}))
	_, err = tw.Write(data)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())

	return f.Name()
}

func TestImportClient_Import_Success(t *testing.T) {
	archivePath := createTestArchive(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		assert.Equal(t, http.MethodPost, r.Method)

		// Verify authorization header
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Verify content type is multipart
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

		// Verify path
		assert.Equal(t, "/organizations/org-1/environments/env-1/agent/import", r.URL.Path)

		// Verify file in multipart body
		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)
		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()
		assert.Equal(t, apiclient.ExportArchiveFileName, header.Filename)

		// Verify it's a valid gzip
		gr, err := gzip.NewReader(file)
		require.NoError(t, err)
		defer gr.Close()
		_, err = io.ReadAll(gr)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewImportClient(server.URL, "test-token", "org-1", "env-1")
	err := c.Import(context.Background(), archivePath)
	assert.NoError(t, err)
}

func TestImportClient_Import_ServerError(t *testing.T) {
	archivePath := createTestArchive(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	c := NewImportClient(server.URL, "test-token", "org-1", "env-1")
	err := c.Import(context.Background(), archivePath)
	assert.Error(t, err)
	var httpErr *HTTPError
	require.True(t, errors.As(err, &httpErr))
	assert.Equal(t, http.StatusInternalServerError, httpErr.StatusCode)
	assert.Contains(t, httpErr.Body, "internal server error")
}

func TestImportClient_Import_413(t *testing.T) {
	archivePath := createTestArchive(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = w.Write([]byte("archive too large"))
	}))
	defer server.Close()

	c := NewImportClient(server.URL, "test-token", "org-1", "env-1")
	err := c.Import(context.Background(), archivePath)
	assert.Error(t, err)
	var httpErr *HTTPError
	require.True(t, errors.As(err, &httpErr))
	assert.Equal(t, http.StatusRequestEntityTooLarge, httpErr.StatusCode)
}

func TestImportClient_Import_InvalidPath(t *testing.T) {
	c := NewImportClient("https://api.testkube.io", "token", "org-1", "env-1")
	err := c.Import(context.Background(), "/nonexistent/path/archive.tar.gz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "opening archive")
}

func TestImportClient_Import_ContextCancellation(t *testing.T) {
	archivePath := createTestArchive(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	c := NewImportClient(server.URL, "test-token", "org-1", "env-1")
	err := c.Import(ctx, archivePath)
	assert.Error(t, err)
}

func TestHTTPError_ErrorString(t *testing.T) {
	err := &HTTPError{StatusCode: 413, Body: "too large"}
	assert.Equal(t, "import failed (HTTP 413): too large", err.Error())
}

func TestHTTPError_ErrorsAs(t *testing.T) {
	// Verify that errors.As can extract HTTPError from a generic error interface
	var baseErr error = &HTTPError{StatusCode: http.StatusRequestEntityTooLarge, Body: "archive too large"}

	var httpErr *HTTPError
	require.True(t, errors.As(baseErr, &httpErr))
	assert.Equal(t, http.StatusRequestEntityTooLarge, httpErr.StatusCode)
	assert.Equal(t, "archive too large", httpErr.Body)
}

func TestHTTPError_ErrorsAs_Negative(t *testing.T) {
	// A plain error should not match HTTPError via errors.As
	plainErr := errors.New("some other error")

	var httpErr *HTTPError
	assert.False(t, errors.As(plainErr, &httpErr))
}

func TestImportClient_Import_401Unauthorized(t *testing.T) {
	archivePath := createTestArchive(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	c := NewImportClient(server.URL, "bad-token", "org-1", "env-1")
	err := c.Import(context.Background(), archivePath)
	assert.Error(t, err)
	var httpErr *HTTPError
	require.True(t, errors.As(err, &httpErr))
	assert.Equal(t, http.StatusUnauthorized, httpErr.StatusCode)
	assert.Contains(t, httpErr.Body, "unauthorized")
}

func TestImportClient_Import_EmptyBaseUrl(t *testing.T) {
	archivePath := createTestArchive(t)

	c := NewImportClient("", "token", "org-1", "env-1")
	err := c.Import(context.Background(), archivePath)
	assert.Error(t, err)
}

func TestImportClient_Import_VerifiesPath(t *testing.T) {
	archivePath := createTestArchive(t)

	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewImportClient(server.URL, "token", "my-org", "my-env")
	err := c.Import(context.Background(), archivePath)
	assert.NoError(t, err)
	assert.Equal(t, "/organizations/my-org/environments/my-env/agent/import", receivedPath)
}

func TestNewImportClient_PathConstruction(t *testing.T) {
	tests := []struct {
		name   string
		orgID  string
		envID  string
		expect string
	}{
		{
			name:   "standard IDs",
			orgID:  "org-123",
			envID:  "env-456",
			expect: "/organizations/org-123/environments/env-456/agent/import",
		},
		{
			name:   "empty IDs",
			orgID:  "",
			envID:  "",
			expect: "/organizations//environments//agent/import",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewImportClient("https://api.example.com", "token", tt.orgID, tt.envID)
			assert.Equal(t, tt.expect, c.Path)
		})
	}
}

func TestImportClient_Import_ErrorResponseTruncation(t *testing.T) {
	archivePath := createTestArchive(t)

	// Generate a response body larger than maxErrorResponseBytes (1 MB)
	largeBody := bytes.Repeat([]byte("x"), 2*1024*1024) // 2 MB

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(largeBody)
	}))
	defer server.Close()

	c := NewImportClient(server.URL, "test-token", "org-1", "env-1")
	err := c.Import(context.Background(), archivePath)
	assert.Error(t, err)
	// Error message should be truncated to maxErrorResponseBytes
	assert.LessOrEqual(t, len(err.Error()), maxErrorResponseBytes+200) // +200 for the prefix
}
