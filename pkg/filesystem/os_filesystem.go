package filesystem

import (
	"bufio"
	"os"
	"path/filepath"
)

type OSFileSystem struct{}

func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

func (s *OSFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (s *OSFileSystem) Walk(root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(root, walkFn)
}

func (s *OSFileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

func (s *OSFileSystem) OpenFileRO(name string) (*os.File, error) {
	return os.Open(name)
}

func (s *OSFileSystem) OpenFileBuffered(name string) (*bufio.Reader, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	return bufio.NewReader(f), nil
}

var _ FileSystem = (*OSFileSystem)(nil)
