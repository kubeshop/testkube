package runner

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Runner interface to abstract runners implementations
type Runner interface {
	// Run takes Execution data and returns execution result
	Run(execution testkube.Execution) testkube.ExecutionResult
}
