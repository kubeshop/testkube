package client

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		file, header, err := r.FormFile("archive")
		require.NoError(t, err)
		defer file.Close()
		assert.Equal(t, "testkube-export.tar.gz", header.Filename)

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
	assert.Contains(t, err.Error(), "HTTP 500")
	assert.Contains(t, err.Error(), "internal server error")
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
	assert.Contains(t, err.Error(), "413")
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
