package tmp

import (
	"io"
	"io/ioutil"
)

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

func Name() string {
	tmpfile, _ := ioutil.TempFile("", "testkube-tmp")
	return tmpfile.Name()
}
