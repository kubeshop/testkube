package newman

import (
	"io"
	"io/ioutil"
	"os/exec"
)

type Runner struct {
}

func (r *Runner) RunCollection(input io.Reader) (result TextResult, err error) {
	path, err := r.SaveToTmpFile(input)
	if err != nil {
		return result, err
	}
	out, err := exec.Command("newman", "run", path).Output()
	return TextResult{Output: out}, err
}

func (r *Runner) SaveToTmpFile(input io.Reader) (path string, err error) {
	tmpfile, err := ioutil.TempFile("", "example")
	path = tmpfile.Name()
	if _, err := io.Copy(tmpfile, input); err != nil {
		return path, err
	}

	if err := tmpfile.Close(); err != nil {
		return path, err
	}

	return
}
