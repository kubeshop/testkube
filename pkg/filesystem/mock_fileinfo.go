package filesystem

import (
	"os"
	"time"
)

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
