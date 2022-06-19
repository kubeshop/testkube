package tmp

import (
	"io"
	"os"
)

// ReaderToTmpfile converts io.Reader to tmp file returns saved file path
func ReaderToTmpfile(input io.Reader) (path string, err error) {
	tmpfile, err := os.CreateTemp("", "testkube-tmp")
	path = tmpfile.Name()
	if _, err := io.Copy(tmpfile, input); err != nil {
		return path, err
	}

	if err := tmpfile.Close(); err != nil {
		return path, err
	}

	return
}

// Name generate new temp file and returns file path
func Name() string {
	tmpfile, _ := os.CreateTemp("", "testkube-tmp")
	return tmpfile.Name()
}
