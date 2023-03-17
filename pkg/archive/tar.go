package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"

	"github.com/pkg/errors"
)

func CreateTarball(files []*os.File, out io.Writer) error {
	gzipWriter := gzip.NewWriter(out)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for i := range files {
		err := addFileToTarWriter(files[i], tarWriter)
		if err != nil {
			return errors.Wrapf(err, "error adding file %s to tarball", files[i].Name())
		}
	}

	return nil
}

func addFileToTarWriter(file *os.File, tarWriter *tar.Writer) error {
	stat, err := file.Stat()
	if err != nil {
		return errors.Wrapf(err, "could not stat file %s", file.Name())
	}
	header := &tar.Header{
		Name:    stat.Name(),
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return errors.Wrapf(err, "error writing header for file %s", file.Name())
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return errors.Wrapf(err, "error copying file %s to tarball", file.Name())
	}

	return nil
}
