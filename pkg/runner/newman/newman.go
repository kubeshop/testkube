package newman

import (
	"io"

	"github.com/kubeshop/kubetest/pkg/api/executor"
	"github.com/kubeshop/kubetest/pkg/process"
	"github.com/kubeshop/kubetest/pkg/tmp"
)

type Runner struct {
}

func (r *Runner) Run(input io.Reader) (result executor.Execution, err error) {
	path, err := tmp.ReaderToTmpfile(input)
	if err != nil {
		return result, err
	}

	out, err := process.Execute("newman", "run", path)
	return executor.Execution{Output: string(out)}, err
}
