package tmp

import (
	"io"
	"io/ioutil"
)

func ReaderToTmpfile(input io.Reader) (path string, err error) {
	tmpfile, err := ioutil.TempFile("", "kubetest-tmp")
	path = tmpfile.Name()
	if _, err := io.Copy(tmpfile, input); err != nil {
		return path, err
	}

	if err := tmpfile.Close(); err != nil {
		return path, err
	}

	return
}
