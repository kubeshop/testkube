package scraper

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/kubeshop/testkube/pkg/archive"
	"github.com/kubeshop/testkube/pkg/log"

	"github.com/kubeshop/testkube/pkg/filesystem"

	"github.com/pkg/errors"
)

type ArchiveFilesystemExtractor struct {
	fs filesystem.FileSystem
}

func NewArchiveFilesystemExtractor(fs filesystem.FileSystem) *ArchiveFilesystemExtractor {
	return &ArchiveFilesystemExtractor{fs: fs}
}

func (e *ArchiveFilesystemExtractor) Extract(ctx context.Context, paths []string, process ProcessFn) error {
	log.DefaultLogger.Infof("extracting files from directories: %v", paths)
	var archiveFiles []*archive.File
	for _, dir := range paths {
		log.DefaultLogger.Debugf("walking directory: %v", dir)
		err := e.fs.Walk(
			dir,
			func(path string, fileInfo os.FileInfo, err error) error {
				log.DefaultLogger.Debugf("walking path %s", path)
				if err != nil {
					return errors.Wrapf(err, "error walking path %s", path)
				}

				if fileInfo.IsDir() {
					log.DefaultLogger.Debugf("skipping directory %s", path)
					return nil
				}

				archiveFile, err := e.newArchiveFile(dir, path)
				if err != nil {
					return errors.Wrapf(err, "error creating archive file for path %s", path)
				}
				archiveFiles = append(archiveFiles, archiveFile)
				return nil
			},
		)

		if err != nil {
			return errors.Wrapf(err, "error walking directory %s", dir)
		}
	}

	tarballService := archive.NewTarballService()
	var artifactsTarball bytes.Buffer
	log.DefaultLogger.Infof("creating artifacts tarball with %d files", len(archiveFiles))
	if err := tarballService.Create(&artifactsTarball, archiveFiles); err != nil {
		return errors.Wrapf(err, "error creating tarball")
	}

	object := &Object{
		Name:     "artifacts.tar.gz",
		Size:     int64(artifactsTarball.Len()),
		Data:     &artifactsTarball,
		DataType: DataTypeTarball,
	}
	if err := process(ctx, object); err != nil {
		return errors.Wrapf(err, "error processing object %s", object.Name)
	}

	return nil
}

func (e *ArchiveFilesystemExtractor) newArchiveFile(baseDir string, path string) (*archive.File, error) {
	f, err := e.fs.OpenFileRO(path)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening file %s", path)
	}

	stat, err := e.fs.Stat(path)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting file stat %s", path)
	}

	relpath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting relative path for %s", path)
	}
	if relpath == "." {
		relpath = stat.Name()
	}

	archiveFile := archive.File{
		Name:    relpath,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
		Data:    &bytes.Buffer{},
	}
	n, err := io.Copy(archiveFile.Data, f)
	if err != nil {
		return nil, errors.Wrapf(err, "error copying file %s data to tarball", path)
	}
	if n != stat.Size() {
		return nil, errors.Errorf("error copying file %s data to tarball, expected %d bytes, got %d", path, stat.Size(), n)
	}

	return &archiveFile, nil
}

var _ Extractor = (*ArchiveFilesystemExtractor)(nil)

type RecursiveFilesystemExtractor struct {
	fs filesystem.FileSystem
}

func NewRecursiveFilesystemExtractor(fs filesystem.FileSystem) *RecursiveFilesystemExtractor {
	return &RecursiveFilesystemExtractor{fs: fs}
}

func (e *RecursiveFilesystemExtractor) Extract(ctx context.Context, paths []string, process ProcessFn) error {
	log.DefaultLogger.Infof("extracting files from directories: %v", paths)
	for _, dir := range paths {
		log.DefaultLogger.Infof("walking directory: %v", dir)

		if _, err := e.fs.Stat(dir); os.IsNotExist(err) {
			log.DefaultLogger.Warnf("directory %s does not exist, skipping", dir)
			continue
		}

		err := e.fs.Walk(
			dir,
			func(path string, fileInfo os.FileInfo, err error) error {
				log.DefaultLogger.Infof("walking path %s", path)
				if err != nil {
					return errors.Wrapf(err, "error walking path %s", path)
				}

				if fileInfo.IsDir() {
					log.DefaultLogger.Infof("skipping directory %s", path)
					return nil
				}

				reader, err := e.fs.OpenFileBuffered(path)
				if err != nil {
					return errors.Wrapf(err, "error opening buffered %s", path)
				}
				relpath, err := filepath.Rel(dir, path)
				if err != nil {
					return errors.Wrapf(err, "error getting relative path for %s", path)
				}
				if relpath == "." {
					relpath = fileInfo.Name()
				}
				object := &Object{
					Name:     relpath,
					Size:     fileInfo.Size(),
					Data:     reader,
					DataType: DataTypeRaw,
				}
				log.DefaultLogger.Infof("filesystem extractor is sending file to be processed: %v", object.Name)
				if err := process(ctx, object); err != nil {
					return errors.Wrapf(err, "failed to process file %s", object.Name)
				}

				return nil
			})
		if err != nil {
			return errors.Wrapf(err, "failed to walk directory %s", dir)
		}
	}

	return nil
}

var _ Extractor = (*RecursiveFilesystemExtractor)(nil)
