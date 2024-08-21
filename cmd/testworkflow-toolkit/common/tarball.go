package common

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
)

var (
	relativeCheckRe = regexp.MustCompile(`(^|/)\.\.(/|$)`)
)

func WriteTarball(stream io.Writer, dirPath string, files []string) error {
	// Ensure the absolute path
	if !filepath.IsAbs(dirPath) {
		var err error
		dirPath, err = filepath.Abs(dirPath)
		if err != nil {
			return errors.Wrap(err, "failed to build absolute path for writing tarball")
		}
	}

	// Prepare files archive
	gzipStream := gzip.NewWriter(stream)
	tarStream := tar.NewWriter(gzipStream)
	defer gzipStream.Close()
	defer tarStream.Close()

	// Append all the files
	walker, err := artifacts.CreateWalker(files, []string{dirPath}, dirPath)
	if err != nil {
		return err
	}
	err = walker.Walk(os.DirFS("/"), func(path string, file fs.File, stat fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Warning: '%s' has been ignored, as there was a problem reading it: %s\n", path, err.Error())
			return nil
		}

		// Append the file to the archive
		name := stat.Name()
		link := name
		isSymlink := stat.Mode()&fs.ModeSymlink != 0
		if isSymlink {
			link, err = os.Readlink(filepath.Join(dirPath, path))
			if err != nil {
				fmt.Printf("Warning: '%s' has been ignored, as there was a problem reading link: %s\n", path, err.Error())
				return nil
			}
		}

		// Build the data
		header, err := tar.FileInfoHeader(stat, link)
		if err != nil {
			return err
		}
		header.Name = path
		err = tarStream.WriteHeader(header)
		if err != nil {
			return err
		}

		// Copy the contents for regular files
		if !isSymlink {
			_, err = io.Copy(tarStream, file)
		}

		return err
	})
	return err
}

func UnpackTarball(dirPath string, stream io.Reader) error {
	// Process the files
	uncompressedStream, err := gzip.NewReader(stream)
	if err != nil {
		return errors.Wrap(err, "start reading gzip")
	}
	tarReader := tar.NewReader(uncompressedStream)

	// Unpack them
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "get next entry from tarball")
		}
		if filepath.IsAbs(header.Name) || relativeCheckRe.MatchString(filepath.ToSlash(header.Name)) {
			return fmt.Errorf("unsafe file path in the tarball: %s", header.Name)
		}

		filePath := filepath.Join(dirPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			err := os.Mkdir(filePath, 0755)
			if err != nil {
				return errors.Wrapf(err, "%s: create directory", filePath)
			}
		case tar.TypeReg:
			err := os.MkdirAll(filepath.Dir(filePath), 0755)
			if err != nil {
				return errors.Wrapf(err, "%s: create directory tree", filePath)
			}
			outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return errors.Wrapf(err, "%s: create file", filePath)
			}
			_, err = io.Copy(outFile, tarReader)
			if err != nil {
				_ = outFile.Close()
				return errors.Wrapf(err, "%s: write file", filePath)
			}
			_ = outFile.Close()
		case tar.TypeSymlink:
			err := os.MkdirAll(filepath.Dir(filePath), 0755)
			if err != nil {
				return errors.Wrapf(err, "%s: create directory tree", filePath)
			}
			err = os.Symlink(header.Linkname, filePath)
			if err != nil {
				return errors.Wrapf(err, "%s: create symlink", filePath)
			}
		default:
			return fmt.Errorf("unknown entry type in the transferred archive: '%x' in %s", header.Typeflag, filePath)
		}
	}
	return nil
}
