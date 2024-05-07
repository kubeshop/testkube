package filesystem

import (
	"bufio"
	"io/fs"
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

func (s *OSFileSystem) OpenFileRO(name string) (fs.File, error) {
	return os.Open(name)
}

func (s *OSFileSystem) OpenFileBuffered(name string) (*bufio.Reader, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	return bufio.NewReader(f), nil
}

func (s *OSFileSystem) ReadFileBuffered(name string) (reader *bufio.Reader, closer func() error, err error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, nil, err
	}

	return bufio.NewReader(f), f.Close, nil
}

func (s *OSFileSystem) ReadDir(dirname string) ([]os.DirEntry, error) {
	return os.ReadDir(dirname)
}

func (s *OSFileSystem) Getwd() (string, error) {
	return os.Getwd()
}

func (s *OSFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

var _ FileSystem = (*OSFileSystem)(nil)
