package scraper

import (
	"context"
	"os"

	"github.com/kubeshop/testkube/pkg/log"

	"github.com/kubeshop/testkube/pkg/filesystem"

	"github.com/pkg/errors"
)

type FilesystemExtractor struct {
	dirs []string
	fs   filesystem.FileSystem
}

func NewFilesystemExtractor(dirs []string, fs filesystem.FileSystem) *FilesystemExtractor {
	return &FilesystemExtractor{dirs: dirs, fs: fs}
}

func (e *FilesystemExtractor) Extract(ctx context.Context, process ProcessFn) error {
	log.DefaultLogger.Infof("extracting files from directories: %v", e.dirs)
	for _, dir := range e.dirs {
		log.DefaultLogger.Infof("walking directory: %v", e.dirs)

		if _, err := e.fs.Stat(dir); os.IsNotExist(err) {
			log.DefaultLogger.Debugw("directory %s does not exist, skipping", dir)
			continue
		}

		err := e.fs.Walk(
			dir,
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
				log.DefaultLogger.Infof("filesystem extractor is sending file to be processed: %v", fileInfo.Name())
				if err := process(ctx, object); err != nil {
					return errors.Wrapf(err, "failed to process file %s", fileInfo.Name())
				}

				return nil
			})
		if err != nil {
			return errors.Wrapf(err, "failed to walk directory %s", dir)
		}
	}

	return nil
}
