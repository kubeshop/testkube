package commands

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessTransferPair(t *testing.T) {
	t.Run("sends tarball successfully", func(t *testing.T) {
		// Create test files
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		// Track received requests
		var receivedBody []byte
		receivedHeaders := make(http.Header)

		// Create mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			receivedHeaders = r.Header
			receivedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		// Process transfer
		output := &bytes.Buffer{}
		exitCode := ProcessTransferPair(tempDir+":test.txt="+server.URL+"/upload", output)

		// Verify success
		assert.Equal(t, 0, exitCode, "should return exit code 0 on success")
		assert.Contains(t, output.String(), "Packing and sending")
		assert.Equal(t, "application/tar+gzip", receivedHeaders.Get("Content-Type"))

		// Verify tarball contents
		assert.Greater(t, len(receivedBody), 0, "should have received tarball data")

		// Unpack and verify
		gzReader, err := gzip.NewReader(bytes.NewReader(receivedBody))
		require.NoError(t, err)
		defer gzReader.Close()

		tarReader := tar.NewReader(gzReader)
		header, err := tarReader.Next()
		require.NoError(t, err)
		assert.Equal(t, "test.txt", header.Name)

		content, err := io.ReadAll(tarReader)
		require.NoError(t, err)
		assert.Equal(t, "test content", string(content))
	})

	t.Run("sends multiple files with patterns", func(t *testing.T) {
		// Create test files
		tempDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content1"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("content2"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tempDir, "file.log"), []byte("log content"), 0644)
		require.NoError(t, err)

		// Track received data
		var receivedBody []byte

		// Create mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		// Process transfer with pattern
		output := &bytes.Buffer{}
		exitCode := ProcessTransferPair(tempDir+":*.txt="+server.URL+"/upload", output)

		// Verify success
		assert.Equal(t, 0, exitCode, "should return exit code 0 on success")

		// Verify tarball contains only txt files
		gzReader, err := gzip.NewReader(bytes.NewReader(receivedBody))
		require.NoError(t, err)
		defer gzReader.Close()

		tarReader := tar.NewReader(gzReader)
		fileCount := 0
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			assert.Contains(t, header.Name, ".txt")
			assert.NotContains(t, header.Name, ".log")
			fileCount++
		}
		assert.Equal(t, 2, fileCount, "should have 2 txt files")
	})

	t.Run("uses default pattern when empty", func(t *testing.T) {
		// Create test files
		tempDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0644)
		require.NoError(t, err)

		// Create mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		// Process transfer with empty pattern
		output := &bytes.Buffer{}
		exitCode := ProcessTransferPair(tempDir+":="+server.URL+"/upload", output)

		// Verify success (default pattern **/* should be used)
		assert.Equal(t, 0, exitCode, "should return exit code 0 with default pattern")
	})

	t.Run("handles server errors", func(t *testing.T) {
		// Create test files
		tempDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0644)
		require.NoError(t, err)

		// Create mock HTTP server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		// Process transfer
		output := &bytes.Buffer{}
		exitCode := ProcessTransferPair(tempDir+":test.txt="+server.URL+"/upload", output)

		// Verify failure
		assert.Equal(t, 1, exitCode, "should return exit code 1 on server error")
		assert.Contains(t, output.String(), "status code 500")
	})

	t.Run("handles connection errors", func(t *testing.T) {
		// Create test files
		tempDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0644)
		require.NoError(t, err)

		// Process transfer to invalid URL
		output := &bytes.Buffer{}
		exitCode := ProcessTransferPair(tempDir+":test.txt=http://localhost:99999/upload", output)

		// Verify failure
		assert.Equal(t, 1, exitCode, "should return exit code 1 on connection error")
		assert.Contains(t, output.String(), "error: send the tarball request")
	})

	t.Run("handles empty directory", func(t *testing.T) {
		// Create mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		// Process transfer with non-existent directory (WriteTarball creates empty tarball)
		output := &bytes.Buffer{}
		exitCode := ProcessTransferPair("/non/existent/dir:*="+server.URL+"/upload", output)

		// Verify success with empty tarball
		assert.Equal(t, 0, exitCode, "should return exit code 0 even with non-existent directory")
		assert.Contains(t, output.String(), "Packing and sending")
	})

	t.Run("returns error on invalid pair format - missing colon", func(t *testing.T) {
		output := &bytes.Buffer{}
		exitCode := ProcessTransferPair("invalid-pair-no-colon", output)

		assert.Equal(t, 1, exitCode, "should return exit code 1 for invalid pair")
		assert.Contains(t, output.String(), "error: invalid files request")
	})

	t.Run("returns error on invalid pair format - missing equals", func(t *testing.T) {
		output := &bytes.Buffer{}
		exitCode := ProcessTransferPair("/path:pattern-no-equals", output)

		assert.Equal(t, 1, exitCode, "should return exit code 1 for invalid pair")
		assert.Contains(t, output.String(), "error: invalid files request")
	})
}

func TestProcessTransfers(t *testing.T) {
	t.Run("returns 0 for empty pairs", func(t *testing.T) {
		output := &bytes.Buffer{}
		exitCode := ProcessTransfers([]string{}, output)

		assert.Equal(t, 0, exitCode, "should return exit code 0 for empty pairs")
		assert.Contains(t, output.String(), "nothing to send")
	})

	t.Run("processes multiple transfers successfully", func(t *testing.T) {
		// Create test files
		tempDir1 := t.TempDir()
		tempDir2 := t.TempDir()
		err := os.WriteFile(filepath.Join(tempDir1, "file1.txt"), []byte("content1"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tempDir2, "file2.txt"), []byte("content2"), 0644)
		require.NoError(t, err)

		requestCount := 0

		// Create mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		// Process transfers
		output := &bytes.Buffer{}
		pairs := []string{
			tempDir1 + ":file1.txt=" + server.URL + "/upload1",
			tempDir2 + ":file2.txt=" + server.URL + "/upload2",
		}
		exitCode := ProcessTransfers(pairs, output)

		// Verify success
		assert.Equal(t, 0, exitCode, "should return exit code 0 when all succeed")
		assert.Equal(t, 2, requestCount, "should have made 2 requests")

		outputStr := output.String()
		assert.Contains(t, outputStr, tempDir1)
		assert.Contains(t, outputStr, tempDir2)
	})

	t.Run("stops on first failure", func(t *testing.T) {
		// Create test files
		tempDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0644)
		require.NoError(t, err)

		requestCount := 0

		// Create mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if r.URL.Path == "/fail" {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		}))
		defer server.Close()

		// Process transfers
		output := &bytes.Buffer{}
		pairs := []string{
			tempDir + ":test.txt=" + server.URL + "/fail",
			tempDir + ":test.txt=" + server.URL + "/success",
		}
		exitCode := ProcessTransfers(pairs, output)

		// Verify failure
		assert.Equal(t, 1, exitCode, "should return exit code 1 on first failure")
		assert.Equal(t, 1, requestCount, "should have made only 1 request")
		assert.Contains(t, output.String(), "status code 500")
	})
}
