package test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/commands"
)

// setupToolkitTest configures the minimal environment required for toolkit tests.
// The toolkit loads configuration during package initialization, before main() is called,
// so we must set these environment variables before importing the commands package.
func setupToolkitTest(t *testing.T) func() {
	// Save current environment
	oldCfg := os.Getenv("TK_CFG")
	oldRef := os.Getenv("TK_REF")
	oldNs := os.Getenv("TK_NS")

	// Set minimal configuration required by toolkit
	// TK_CFG: JSON configuration normally provided by Kubernetes
	// TK_REF: Step reference identifier
	// TK_NS: Namespace
	os.Setenv("TK_CFG", `{"execution":{"id":"test"},"workflow":{"name":"test"},"worker":{"namespace":"test"}}`)
	os.Setenv("TK_REF", "test")
	os.Setenv("TK_NS", "test")

	// Return cleanup function
	return func() {
		if oldCfg == "" {
			os.Unsetenv("TK_CFG")
		} else {
			os.Setenv("TK_CFG", oldCfg)
		}
		if oldRef == "" {
			os.Unsetenv("TK_REF")
		} else {
			os.Setenv("TK_REF", oldRef)
		}
		if oldNs == "" {
			os.Unsetenv("TK_NS")
		} else {
			os.Setenv("TK_NS", oldNs)
		}
	}
}

func TestProcessTarballPair(t *testing.T) {
	cleanup := setupToolkitTest(t)
	defer cleanup()

	t.Run("downloads and extracts tarball successfully", func(t *testing.T) {
		// Create test tarball content
		tarballData := createTestTarball(t)

		// Create mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/test.tar.gz" {
				w.Header().Set("Content-Type", "application/gzip")
				w.Write(tarballData)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		// Create temp directory for extraction
		tempDir := t.TempDir()
		extractPath := filepath.Join(tempDir, "extracted")

		// Process tarball
		output := &bytes.Buffer{}
		exitCode := commands.ProcessTarballPair(extractPath+"="+server.URL+"/test.tar.gz", output)

		// Verify success
		assert.Equal(t, 0, exitCode, "should return exit code 0 on success")

		// Verify extraction
		assert.DirExists(t, extractPath)
		assert.FileExists(t, filepath.Join(extractPath, "test.txt"))

		// Verify file content
		content, err := os.ReadFile(filepath.Join(extractPath, "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, "test content\n", string(content))

		// Check output
		outputStr := output.String()
		assert.Contains(t, outputStr, "Downloading and unpacking")
	})

	t.Run("handles download failures with retry", func(t *testing.T) {
		attemptCount := 0

		// Create mock HTTP server that fails first then succeeds
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(createTestTarball(t))
		}))
		defer server.Close()

		// Create temp directory
		tempDir := t.TempDir()
		extractPath := filepath.Join(tempDir, "extracted")

		// Process tarball
		output := &bytes.Buffer{}
		exitCode := commands.ProcessTarballPair(extractPath+"="+server.URL+"/retry.tar.gz", output)

		// Verify success after retry
		assert.Equal(t, 0, exitCode, "should return exit code 0 after successful retry")
		assert.Equal(t, 3, attemptCount, "should have made 3 attempts")

		// Check output shows retry messages
		outputStr := output.String()
		assert.Contains(t, outputStr, "retrying")
		assert.Contains(t, outputStr, "attempt 2/5")
		assert.Contains(t, outputStr, "attempt 3/5")

		// Verify extraction succeeded
		assert.DirExists(t, extractPath)
		assert.FileExists(t, filepath.Join(extractPath, "test.txt"))
	})

	t.Run("returns error on max retries exceeded", func(t *testing.T) {
		// Create mock HTTP server that always fails
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		// Create temp directory
		tempDir := t.TempDir()
		extractPath := filepath.Join(tempDir, "extracted")

		// Process tarball
		output := &bytes.Buffer{}
		exitCode := commands.ProcessTarballPair(extractPath+"="+server.URL+"/fail.tar.gz", output)

		// Verify failure
		assert.Equal(t, 1, exitCode, "should return exit code 1 after max retries")

		// Check output shows all retry attempts
		outputStr := output.String()
		assert.Contains(t, outputStr, "attempt 2/5")
		assert.Contains(t, outputStr, "attempt 3/5")
		assert.Contains(t, outputStr, "attempt 4/5")
		assert.Contains(t, outputStr, "attempt 5/5")
		assert.Contains(t, outputStr, "failed to download")
	})

	t.Run("returns error on invalid pair format", func(t *testing.T) {
		output := &bytes.Buffer{}
		exitCode := commands.ProcessTarballPair("invalid-pair-no-equals", output)

		assert.Equal(t, 1, exitCode, "should return exit code 1 for invalid pair")
		assert.Contains(t, output.String(), "invalid tarball pair format")
	})
}

func TestProcessTarballs(t *testing.T) {
	cleanup := setupToolkitTest(t)
	defer cleanup()

	t.Run("returns 0 for empty pairs", func(t *testing.T) {
		output := &bytes.Buffer{}
		exitCode := commands.ProcessTarballs([]string{}, output)

		assert.Equal(t, 0, exitCode, "should return exit code 0 for empty pairs")
		assert.Contains(t, output.String(), "nothing to fetch and unpack")
	})

	t.Run("processes multiple tarballs successfully", func(t *testing.T) {
		// Create mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(createTestTarball(t))
		}))
		defer server.Close()

		// Create temp directory
		tempDir := t.TempDir()
		pairs := []string{
			filepath.Join(tempDir, "dir1") + "=" + server.URL + "/tar1.tar.gz",
			filepath.Join(tempDir, "dir2") + "=" + server.URL + "/tar2.tar.gz",
		}

		// Process tarballs
		output := &bytes.Buffer{}
		exitCode := commands.ProcessTarballs(pairs, output)

		// Verify success
		assert.Equal(t, 0, exitCode, "should return exit code 0 when all succeed")

		// Verify both extractions
		assert.FileExists(t, filepath.Join(tempDir, "dir1", "test.txt"))
		assert.FileExists(t, filepath.Join(tempDir, "dir2", "test.txt"))
	})

	t.Run("stops on first failure", func(t *testing.T) {
		// Create mock HTTP server
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if r.URL.Path == "/fail.tar.gz" {
				// Always fail for this one
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.Header().Set("Content-Type", "application/gzip")
				w.Write(createTestTarball(t))
			}
		}))
		defer server.Close()

		tempDir := t.TempDir()
		pairs := []string{
			filepath.Join(tempDir, "dir1") + "=" + server.URL + "/fail.tar.gz",
			filepath.Join(tempDir, "dir2") + "=" + server.URL + "/ok.tar.gz",
		}

		// Process tarballs
		output := &bytes.Buffer{}
		exitCode := commands.ProcessTarballs(pairs, output)

		// Verify failure
		assert.Equal(t, 1, exitCode, "should return exit code 1 on first failure")

		// The second tarball should not be attempted
		assert.NoDirExists(t, filepath.Join(tempDir, "dir2"))
	})
}

// createTestTarball creates a simple tarball for testing
func createTestTarball(t *testing.T) []byte {
	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	// Add test.txt file to the tar
	content := []byte("test content\n")
	header := &tar.Header{
		Name:     "test.txt",
		Mode:     0644,
		Size:     int64(len(content)),
		ModTime:  time.Now(),
		Typeflag: tar.TypeReg,
	}

	err := tarWriter.WriteHeader(header)
	require.NoError(t, err, "failed to write tar header")

	_, err = tarWriter.Write(content)
	require.NoError(t, err, "failed to write tar content")

	// Close writers in correct order
	err = tarWriter.Close()
	require.NoError(t, err, "failed to close tar writer")

	err = gzipWriter.Close()
	require.NoError(t, err, "failed to close gzip writer")

	return buf.Bytes()
}
