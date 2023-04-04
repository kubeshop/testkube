package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type Tarball struct{}

func NewTarballService() *Tarball {
	return &Tarball{}
}

func (t *Tarball) Extract(in io.Reader) ([]*File, error) {
	tarReader, err := GetTarballReader(in)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating tarball reader")
	}

	var files []*File
	for true {
		header, err := tarReader.Next()

		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.Wrapf(err, "error reading next tarball header")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// do nothing
		case tar.TypeReg:
			sanitizedFilepath := t.sanitizeFilepath(header.Name)
			file := &File{Name: sanitizedFilepath, Size: header.Size, Mode: header.Mode, ModTime: header.ModTime, Data: new(bytes.Buffer)}
			if _, err := io.Copy(file.Data, tarReader); err != nil {
				return nil, errors.Wrapf(err, "error copying file %s data to tarball", file.Name)
			}
			files = append(files, file)
		default:
			return nil, errors.Errorf("uknown %v type in tarball %s", header.Typeflag, header.Name)
		}
	}
	return files, nil
}

func (t *Tarball) sanitizeFilepath(path string) string {
	cleaned := filepath.Clean(path)
	if strings.HasPrefix(cleaned, "..") {
		cleaned = strings.TrimPrefix(cleaned, "..")
	}
	if strings.HasPrefix(cleaned, "/") {
		cleaned = strings.TrimPrefix(cleaned, "/")
	}
	return cleaned
}

func (t *Tarball) Create(out io.Writer, files []*File) (*Meta, error) {
	gzipWriter := gzip.NewWriter(out)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	totalSize := int64(0)
	for _, file := range files {
		size, err := t.addFileToTarWriter(file, tarWriter)
		if err != nil {
			return nil, errors.Wrapf(err, "error adding file %s to tarball", file.Name)
		}
		totalSize += size
	}

	return &Meta{Size: totalSize}, nil
}

func (t *Tarball) addFileToTarWriter(file *File, tarWriter *tar.Writer) (size int64, err error) {
	tarHeader := &tar.Header{Name: file.Name, Mode: file.Mode, Size: file.Size, ModTime: file.ModTime}
	if err := tarWriter.WriteHeader(tarHeader); err != nil {
		return 0, errors.Wrapf(err, "error writing header for file %s in tarball", file.Name)
	}

	n, err := io.Copy(tarWriter, file.Data)
	if err != nil {
		return 0, errors.Wrapf(err, "error copying file %s data to tarball", file.Name)
	}

	return n, nil
}

func ExtractTarballToFS(gzipStream io.Reader, destinationDir string) error {
	tarReader, err := GetTarballReader(gzipStream)
	if err != nil {
		return errors.Wrapf(err, "error creating tarball reader")
	}
	for true {
		header, err := tarReader.Next()

		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return errors.Wrapf(err, "error reading next tarball header")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			path := filepath.Join(destinationDir, header.Name)
			if err := os.Mkdir(path, 0755); err != nil {
				return errors.Wrapf(err, "error creating directory %s", path)
			}
		case tar.TypeReg:
			path := filepath.Join(destinationDir, header.Name)
			outFile, err := os.Create(path)
			if err != nil {
				return errors.Wrapf(err, "error creating file %s", path)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return errors.Wrapf(err, "error copying file %s data to tarball", path)
			}
			outFile.Close()
		default:
			return errors.Errorf("uknown %v type in tarball %s", header.Typeflag, header.Name)
		}
	}
	return nil
}

func GetTarballReader(gzipStream io.Reader) (*tar.Reader, error) {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating gzip reader")
	}

	return tar.NewReader(uncompressedStream), nil
}

var _ Archive = (*Tarball)(nil)
