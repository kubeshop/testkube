package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupCertAuth(t *testing.T) {
	t.Run("no certs returns empty args and cleanups", func(t *testing.T) {
		opts := &CloneOptions{}
		args, cleanups, err := setupCertAuth(opts)
		require.NoError(t, err)
		assert.Empty(t, args)
		assert.Empty(t, cleanups)
	})

	t.Run("CA cert writes temp file and sets sslCAInfo", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)

		opts := &CloneOptions{CaCert: "ca-cert-content"}
		args, cleanups, err := setupCertAuth(opts)
		require.NoError(t, err)
		require.Len(t, cleanups, 1)
		defer RunCleanupFuncs(cleanups)

		require.Len(t, args, 2)
		assert.Equal(t, "-c", args[0])
		assert.True(t, strings.HasPrefix(args[1], "http.sslCAInfo="))

		caPath := strings.TrimPrefix(args[1], "http.sslCAInfo=")
		content, err := os.ReadFile(caPath)
		require.NoError(t, err)
		assert.Equal(t, "ca-cert-content", string(content))
	})

	t.Run("client cert writes temp file and sets sslCert", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)

		opts := &CloneOptions{ClientCert: "client-cert-content"}
		args, cleanups, err := setupCertAuth(opts)
		require.NoError(t, err)
		require.Len(t, cleanups, 1)
		defer RunCleanupFuncs(cleanups)

		require.Len(t, args, 2)
		assert.Equal(t, "-c", args[0])
		assert.True(t, strings.HasPrefix(args[1], "http.sslCert="))

		certPath := strings.TrimPrefix(args[1], "http.sslCert=")
		content, err := os.ReadFile(certPath)
		require.NoError(t, err)
		assert.Equal(t, "client-cert-content", string(content))
	})

	t.Run("client key writes temp file and sets sslKey", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)

		opts := &CloneOptions{ClientKey: "client-key-content"}
		args, cleanups, err := setupCertAuth(opts)
		require.NoError(t, err)
		require.Len(t, cleanups, 1)
		defer RunCleanupFuncs(cleanups)

		require.Len(t, args, 2)
		assert.Equal(t, "-c", args[0])
		assert.True(t, strings.HasPrefix(args[1], "http.sslKey="))

		keyPath := strings.TrimPrefix(args[1], "http.sslKey=")
		content, err := os.ReadFile(keyPath)
		require.NoError(t, err)
		assert.Equal(t, "client-key-content", string(content))
	})

	t.Run("all certs set returns all git config args", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)

		opts := &CloneOptions{
			CaCert:     "ca-content",
			ClientCert: "cert-content",
			ClientKey:  "key-content",
		}
		args, cleanups, err := setupCertAuth(opts)
		require.NoError(t, err)
		require.Len(t, cleanups, 3)
		defer RunCleanupFuncs(cleanups)

		// Should have 6 args: 3 pairs of "-c" + "http.sslXxx=path"
		require.Len(t, args, 6)
		assert.Equal(t, "-c", args[0])
		assert.True(t, strings.HasPrefix(args[1], "http.sslCAInfo="))
		assert.Equal(t, "-c", args[2])
		assert.True(t, strings.HasPrefix(args[3], "http.sslCert="))
		assert.Equal(t, "-c", args[4])
		assert.True(t, strings.HasPrefix(args[5], "http.sslKey="))
	})

	t.Run("temp files are removed by cleanup", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)

		opts := &CloneOptions{
			CaCert:     "ca-content",
			ClientCert: "cert-content",
			ClientKey:  "key-content",
		}
		args, cleanups, err := setupCertAuth(opts)
		require.NoError(t, err)
		require.Len(t, cleanups, 3)

		// Collect file paths before cleanup
		paths := make([]string, 0, 3)
		for i := 1; i < len(args); i += 2 {
			parts := strings.SplitN(args[i], "=", 2)
			require.Len(t, parts, 2)
			paths = append(paths, parts[1])
		}

		// Verify files exist
		for _, p := range paths {
			_, err := os.Stat(p)
			require.NoError(t, err, "temp file should exist before cleanup: %s", p)
		}

		// Run cleanup
		RunCleanupFuncs(cleanups)

		// Verify files are removed
		for _, p := range paths {
			_, err := os.Stat(p)
			assert.True(t, os.IsNotExist(err), "temp file should be removed after cleanup: %s", p)
		}
	})

	t.Run("temp files have read-only permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)

		opts := &CloneOptions{CaCert: "ca-content"}
		args, cleanups, err := setupCertAuth(opts)
		require.NoError(t, err)
		defer RunCleanupFuncs(cleanups)

		caPath := strings.TrimPrefix(args[1], "http.sslCAInfo=")
		info, err := os.Stat(caPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0400), info.Mode().Perm())
	})
}

func TestWriteTempCertFile(t *testing.T) {
	t.Run("creates file with correct content", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)

		path, cleanup, err := writeTempCertFile("test-cert-content", "test-cert-*")
		require.NoError(t, err)
		defer cleanup()

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "test-cert-content", string(content))
	})

	t.Run("file has restricted permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)

		path, cleanup, err := writeTempCertFile("test-content", "test-perm-*")
		require.NoError(t, err)
		defer cleanup()

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0400), info.Mode().Perm())
	})

	t.Run("file is in tmp directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)

		path, cleanup, err := writeTempCertFile("content", "test-location-*")
		require.NoError(t, err)
		defer cleanup()

		assert.True(t, strings.HasPrefix(filepath.Base(path), "test-location-"),
			"file should match pattern prefix")
	})

	t.Run("cleanup removes file", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("TMPDIR", tmpDir)

		path, cleanup, err := writeTempCertFile("content", "test-cleanup-*")
		require.NoError(t, err)

		// Verify file exists before cleanup
		_, err = os.Stat(path)
		require.NoError(t, err)

		// Run cleanup
		cleanup()

		// Verify file is removed
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestRunCleanupFuncs(t *testing.T) {
	t.Run("runs all cleanup functions", func(t *testing.T) {
		count := 0
		cleanups := []func(){
			func() { count++ },
			func() { count++ },
			func() { count++ },
		}
		RunCleanupFuncs(cleanups)
		assert.Equal(t, 3, count)
	})

	t.Run("no-op for empty slice", func(t *testing.T) {
		// Should not panic
		RunCleanupFuncs(nil)
		RunCleanupFuncs([]func(){})
	})
}
