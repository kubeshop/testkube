package runner

import (
	"io"

	"github.com/kubeshop/kubetest/pkg/api/executor"
)

type Runner interface {
	Run(io.Reader) (executor.Execution, error)
}
