package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdjustFilePermissions(t *testing.T) {
	t.Run("successful permission adjustment", func(t *testing.T) {
		// Create temporary directory structure
		tmpDir := t.TempDir()

		// Create test files with restrictive permissions
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0400) // read-only
		require.NoError(t, err)

		// Create subdirectory with file
		subDir := filepath.Join(tmpDir, "subdir")
		err = os.Mkdir(subDir, 0700)
		require.NoError(t, err)

		subFile := filepath.Join(subDir, "subtest.txt")
		err = os.WriteFile(subFile, []byte("test"), 0400)
		require.NoError(t, err)

		// Adjust permissions
		err = adjustFilePermissions(tmpDir)
		require.NoError(t, err)

		// Check permissions were adjusted
		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.True(t, info.Mode().Perm()&0o060 == 0o060, "File should have group read/write permissions")

		info, err = os.Stat(subFile)
		require.NoError(t, err)
		assert.True(t, info.Mode().Perm()&0o060 == 0o060, "Subdir file should have group read/write permissions")
	})

	t.Run("non-existent directory", func(t *testing.T) {
		err := adjustFilePermissions("/non/existent/path")
		assert.Error(t, err)
	})

	t.Run("handles chmod errors gracefully", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Cannot test permission denied as root")
		}

		// Create a directory structure where we can't change permissions
		tmpDir := t.TempDir()

		// Create a file
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)

		// Make the parent directory read-only
		err = os.Chmod(tmpDir, 0555)
		require.NoError(t, err)
		defer os.Chmod(tmpDir, 0755) // Restore for cleanup

		// adjustFilePermissions should not fail, just log warnings
		err = adjustFilePermissions(tmpDir)
		// The function should complete without error even if chmod fails
		assert.NoError(t, err)
	})
}

func TestCopyDirContents(t *testing.T) {
	// Create source directory with content
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create test file in source
	testFile := filepath.Join(srcDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create subdirectory with file
	subDir := filepath.Join(srcDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	subFile := filepath.Join(subDir, "subtest.txt")
	err = os.WriteFile(subFile, []byte("sub content"), 0644)
	require.NoError(t, err)

	// Copy contents
	err = copyDirContents(srcDir, destDir)
	require.NoError(t, err)

	// Verify files were copied
	destFile := filepath.Join(destDir, "test.txt")
	content, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	destSubFile := filepath.Join(destDir, "subdir", "subtest.txt")
	content, err = os.ReadFile(destSubFile)
	require.NoError(t, err)
	assert.Equal(t, "sub content", string(content))

	// Note: copyRepositoryContents intentionally swallows errors in OnError callback
	// to continue copying even if some files fail. This is by design.
}
