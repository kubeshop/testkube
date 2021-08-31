package runner

import (
	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

// Runner interface to abstract runners implementations
type Runner interface {
	// Run takes Execution data and returns result, can take additional params map
	Run(execution kubtest.Execution, params map[string]string) kubtest.ExecutionResult
}
