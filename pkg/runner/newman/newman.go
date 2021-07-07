package newman

import (
	"io"

	"github.com/kubeshop/kubetest/pkg/process"
	"github.com/kubeshop/kubetest/pkg/runner"
	"github.com/kubeshop/kubetest/pkg/tmp"
)

type Runner struct {
}

func (r *Runner) Run(input io.Reader) (result runner.Result, err error) {
	path, err := tmp.ReaderToTmpfile(input)
	if err != nil {
		return result, err
	}

	out, err := process.Execute("newman", "run", path)
	return runner.Result{Output: string(out)}, err
}
