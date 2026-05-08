package runner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
)

// Mock stdout for testing
type mockStdout struct {
	hints          []string
	sensitiveWords []string
}

func (m *mockStdout) Hint(name, instruction string) {
	m.hints = append(m.hints, fmt.Sprintf("%s:%s", name, instruction))
}

func (m *mockStdout) SetSensitiveWords(words []string) {
	m.sensitiveWords = words
}

// Mock stdoutUnsafe for testing
type mockStdoutUnsafe struct {
	printed []string
	errors  []string
}

func (m *mockStdoutUnsafe) Print(s string) {
	m.printed = append(m.printed, s)
}

func (m *mockStdoutUnsafe) Error(s string) {
	m.errors = append(m.errors, s)
}

// newStateManagerWithFS creates a new state manager with custom filesystem (for testing)
func newStateManagerWithFS(stdout, stdoutUnsafe interface{}, fs FileSystem) StateManager {
	return &stateManager{
		stdout: stdout.(interface {
			Hint(string, string)
			SetSensitiveWords([]string)
		}),
		stdoutUnsafe: stdoutUnsafe.(interface {
			Print(string)
			Error(string)
		}),
		fs: fs,
	}
}

// Mock filesystem for testing
type mockFileSystem struct {
	statFunc      func(name string) (os.FileInfo, error)
	mkdirAllFunc  func(path string, perm os.FileMode) error
	writeFileFunc func(name string, data []byte, perm os.FileMode) error
	chmodFunc     func(name string, mode os.FileMode) error
}

func (m *mockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.statFunc != nil {
		return m.statFunc(name)
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if m.mkdirAllFunc != nil {
		return m.mkdirAllFunc(path, perm)
	}
	return nil
}

func (m *mockFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	if m.writeFileFunc != nil {
		return m.writeFileFunc(name, data, perm)
	}
	return nil
}

func (m *mockFileSystem) Chmod(name string, mode os.FileMode) error {
	if m.chmodFunc != nil {
		return m.chmodFunc(name, mode)
	}
	return nil
}

// Mock file info
type mockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m mockFileInfo) ModTime() time.Time { return m.modTime }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

func TestStateManager_EnsureStateFile(t *testing.T) {
	// Save original state path
	originalPath := constants.StatePath
	defer func() {
		constants.StatePath = originalPath
	}()

	t.Run("creates state file when it doesn't exist", func(t *testing.T) {
		// Setup
		constants.StatePath = "/test/.tktw/state"

		stdout := &mockStdout{}
		stdoutUnsafe := &mockStdoutUnsafe{}

		mkdirCalled := false
		writeFileCalled := false
		chmodCalled := false

		fs := &mockFileSystem{
			statFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			mkdirAllFunc: func(path string, perm os.FileMode) error {
				mkdirCalled = true
				assert.Equal(t, "/test/.tktw", path)
				assert.Equal(t, os.FileMode(0777), perm)
				return nil
			},
			writeFileFunc: func(name string, data []byte, perm os.FileMode) error {
				writeFileCalled = true
				assert.Equal(t, constants.StatePath, name)
				assert.Nil(t, data)
				assert.Equal(t, os.FileMode(0777), perm)
				return nil
			},
			chmodFunc: func(name string, mode os.FileMode) error {
				chmodCalled = true
				assert.Equal(t, constants.StatePath, name)
				assert.Equal(t, os.FileMode(0777), mode)
				return nil
			},
		}

		sm := newStateManagerWithFS(stdout, stdoutUnsafe, fs)

		// Execute
		err := sm.EnsureStateFile()

		// Assert
		require.NoError(t, err)
		assert.True(t, mkdirCalled)
		assert.True(t, writeFileCalled)
		assert.True(t, chmodCalled)

		// Verify output
		assert.Contains(t, stdoutUnsafe.printed, "Creating state...")
		assert.Contains(t, stdoutUnsafe.printed, " done\n")
		assert.Contains(t, stdout.hints, "tktw-init:start")
	})

	t.Run("succeeds when state file already exists", func(t *testing.T) {
		// Setup
		constants.StatePath = "/test/state"

		stdout := &mockStdout{}
		stdoutUnsafe := &mockStdoutUnsafe{}

		fs := &mockFileSystem{
			statFunc: func(name string) (os.FileInfo, error) {
				return mockFileInfo{name: "state", mode: 0644}, nil
			},
		}

		sm := newStateManagerWithFS(stdout, stdoutUnsafe, fs)

		// Execute
		err := sm.EnsureStateFile()

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stdoutUnsafe.printed) // No output when file exists
		assert.Empty(t, stdout.hints)
	})

	t.Run("handles directory creation error", func(t *testing.T) {
		// Setup
		constants.StatePath = "/test/.tktw/state"

		stdout := &mockStdout{}
		stdoutUnsafe := &mockStdoutUnsafe{}

		fs := &mockFileSystem{
			statFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			mkdirAllFunc: func(path string, perm os.FileMode) error {
				return errors.New("permission denied")
			},
		}

		sm := newStateManagerWithFS(stdout, stdoutUnsafe, fs)

		// Execute
		err := sm.EnsureStateFile()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create directory")
		assert.Contains(t, err.Error(), "permission denied")
		assert.Contains(t, stdoutUnsafe.errors, " error\n")
	})

	t.Run("handles file creation error", func(t *testing.T) {
		// Setup
		constants.StatePath = "/test/.tktw/state"

		stdout := &mockStdout{}
		stdoutUnsafe := &mockStdoutUnsafe{}

		fs := &mockFileSystem{
			statFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			mkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			writeFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return errors.New("disk full")
			},
		}

		sm := newStateManagerWithFS(stdout, stdoutUnsafe, fs)

		// Execute
		err := sm.EnsureStateFile()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create state file")
		assert.Contains(t, err.Error(), "disk full")
		assert.Contains(t, stdoutUnsafe.errors, " error\n")
	})

	t.Run("handles stat error", func(t *testing.T) {
		// Setup
		constants.StatePath = "/test/state"

		stdout := &mockStdout{}
		stdoutUnsafe := &mockStdoutUnsafe{}

		fs := &mockFileSystem{
			statFunc: func(name string) (os.FileInfo, error) {
				return nil, errors.New("i/o error")
			},
		}

		sm := newStateManagerWithFS(stdout, stdoutUnsafe, fs)

		// Execute
		err := sm.EnsureStateFile()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot access state file")
		assert.Contains(t, err.Error(), "i/o error")
		assert.Contains(t, stdoutUnsafe.errors, " error\n")
		assert.Contains(t, stdout.hints, "tktw-init:start")
	})
}

func TestStateManager_LoadInitialState_Integration(t *testing.T) {
	t.Run("integration test with real filesystem", func(t *testing.T) {
		// This test uses the real StateManager with actual file operations
		tempDir := t.TempDir()
		originalPath := constants.StatePath
		constants.StatePath = filepath.Join(tempDir, ".tktw", "state")
		defer func() {
			constants.StatePath = originalPath
		}()

		stdout := &mockStdout{}
		stdoutUnsafe := &mockStdoutUnsafe{}

		sm := NewStateManager(stdout, stdoutUnsafe)

		// First ensure the state file exists
		err := sm.EnsureStateFile()
		require.NoError(t, err)

		// Verify file was created
		info, err := os.Stat(constants.StatePath)
		require.NoError(t, err)
		assert.False(t, info.IsDir())
		assert.Equal(t, os.FileMode(0777), info.Mode().Perm())

		// Note: LoadInitialState depends on global orchestration.Setup
		// which we cannot easily mock, so we skip testing it here
		// In a real refactoring, we would inject these dependencies
	})
}
