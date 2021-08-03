package newman

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/kubeshop/kubetest/pkg/process"
	"github.com/kubeshop/kubetest/pkg/tmp"
)

// Runner struct for newman based runner
type Runner struct {
}

// Run runs particular script content on top of newman binary
func (r *Runner) Run(input io.Reader, params map[string]string) (string, error) {
	path, err := tmp.ReaderToTmpfile(input)
	if err != nil {
		return "", err
	}

	// write params to tmp file
	envReader, err := NewEnvFileReader(params)
	if err != nil {
		return "", err
	}
	envpath, err := tmp.ReaderToTmpfile(envReader)
	if err != nil {
		return "", err
	}

	fmt.Printf("%+v\n", params)
	fmt.Printf("%+v\n", envpath)
	f, _ := ioutil.ReadFile(envpath)
	fmt.Printf("%+v\n", string(f))

	out, err := process.Execute("newman", "run", path, "-e", envpath)
	return string(out), err
}
