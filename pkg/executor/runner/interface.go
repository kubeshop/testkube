package runner

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Type describes a type of the runner
type Type string

const (
	// TypeInit is an initialization runner
	TypeInit Type = "init"
	// TypeMain is a main runner
	TypeMain Type = "main"
	// TypeFin is a finalization runner
	TypeFin Type = "finalize"
)

// IsInit if type is init
func (t Type) IsInit() bool {
	return t == TypeInit
}

// IsMain if type is main
func (t Type) IsMain() bool {
	return t == TypeMain
}

// IsFin if type is fin
func (t Type) IsFin() bool {
	return t == TypeFin
}

// Runner interface to abstract runners implementations
type Runner interface {
	// Run takes Execution data and returns execution result
	Run(ctx context.Context, execution testkube.Execution) (result testkube.ExecutionResult, err error)
	// GetType returns runner type
	GetType() Type
}
