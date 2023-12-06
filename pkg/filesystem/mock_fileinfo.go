package filesystem

import (
	"os"
	"time"
)

// MockDirEntry implements os.DirEntry interface for mocking in tests.
type MockDirEntry struct {
	FName  string
	FIsDir bool
}

// Name returns the mocked name of the directory entry.
func (m *MockDirEntry) Name() string {
	return m.FName
}

// IsDir returns the mocked directory flag.
func (m *MockDirEntry) IsDir() bool {
	return m.FIsDir
}

// Type returns the mocked file mode.
func (m *MockDirEntry) Type() os.FileMode {
	if m.FIsDir {
		return os.ModeDir
	}
	return 0
}

// Info returns the mocked file info.
func (m *MockDirEntry) Info() (os.FileInfo, error) {
	return &MockFileInfo{
		FName:  m.FName,
		FIsDir: true,
	}, nil
}

type MockFileInfo struct {
	FName    string
	FSize    int64
	FMode    os.FileMode
	FModTime time.Time
	FIsDir   bool
}

func (m *MockFileInfo) Name() string {
	return m.FName
}

func (m *MockFileInfo) Size() int64 {
	return m.FSize
}

func (m *MockFileInfo) Mode() os.FileMode {
	return m.FMode
}

func (m *MockFileInfo) ModTime() time.Time {
	return m.FModTime
}

func (m *MockFileInfo) IsDir() bool {
	return m.FIsDir
}

func (m *MockFileInfo) Sys() interface{} {
	return nil
}
