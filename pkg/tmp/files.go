package tmp

import (
	"io"
	"io/ioutil"
)

// ReaderToTmpfile converts io.Reader to tmp file returns saved file path
func ReaderToTmpfile(input io.Reader) (path string, err error) {
	tmpfile, err := ioutil.TempFile("", "testkube-tmp")
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
	tmpfile, _ := ioutil.TempFile("", "testkube-tmp")
	return tmpfile.Name()
}
