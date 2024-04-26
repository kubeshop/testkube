package filesystem

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
)

//go:generate mockgen -destination=./mock_filesystem.go -package=filesystem "github.com/kubeshop/testkube/pkg/filesystem" FileSystem
type FileSystem interface {
	Stat(name string) (os.FileInfo, error)
	Walk(root string, walkFn filepath.WalkFunc) error
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
	OpenFileRO(name string) (fs.File, error)
	// Deprecated: Use ReadFileBuffered instead. This implementation does not close the file.
	OpenFileBuffered(name string) (*bufio.Reader, error)
	ReadDir(dirname string) ([]os.DirEntry, error)
	ReadFile(filename string) ([]byte, error)
	// ReadFileBuffered returns a buffered reader and a close function for the file.
	ReadFileBuffered(filename string) (reader *bufio.Reader, closer func() error, err error)
	Getwd() (string, error)
}
