package scraper

import (
	"context"
	"os"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/internal/common/filesystem"
)

type FilesystemExtractor struct {
	dir string
	fs  filesystem.FileSystem
}

func NewFilesystemExtractor(dir string, fs filesystem.FileSystem) *FilesystemExtractor {
	return &FilesystemExtractor{dir: dir, fs: fs}
}

func (e *FilesystemExtractor) Extract(ctx context.Context, process ProcessFn) error {
	err := e.fs.Walk(
		e.dir,
		func(path string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrapf(err, "error walking path %s", path)
			}

			if fileInfo.IsDir() {
				return nil
			}

			reader, err := e.fs.OpenFileBuffered(path)
			if err != nil {
				return errors.Wrapf(err, "error opening buffered %s", path)
			}
			object := &Object{
				Name: fileInfo.Name(),
				Size: fileInfo.Size(),
				Data: reader,
			}
			if err := process(ctx, object); err != nil {
				return errors.Wrapf(err, "failed to process file %s", fileInfo.Name())
			}

			return nil
		})
	if err != nil {
		return errors.Wrapf(err, "failed to walk directory %s", e.dir)
	}

	return nil
}
