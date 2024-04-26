package filesystem

import (
	"bytes"
	"io"
	"io/fs"
	"time"
)

// MockFile implements the fs.File interface for testing purposes.
type MockFile struct {
	name    string
	content *bytes.Reader
}

// NewMockFile creates a new instance of MockFile with the given content.
func NewMockFile(name string, content []byte) *MockFile {
	return &MockFile{
		name:    name,
		content: bytes.NewReader(content),
	}
}

// Stat returns the FileInfo for the file.
func (f *MockFile) Stat() (fs.FileInfo, error) {
	return &MockFileInfo{
		FName:    f.name,
		FSize:    int64(f.content.Len()),
		FModTime: time.Now(),
	}, nil
}

// Read implements the Read method of the io.Reader interface.
func (f *MockFile) Read(b []byte) (int, error) {
	return f.content.Read(b)
}

// Close implements the Close method of the io.Closer interface.
func (f *MockFile) Close() error {
	// You can implement this to simulate closing a file, for example:
	// Reset the reader to simulate a fresh state if reopened.
	_, err := f.content.Seek(0, io.SeekStart)
	return err
}
