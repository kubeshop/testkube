package scheduling

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type PostgresExecutionQuerier struct{}

func NewPostgresExecutionQuerier() *PostgresExecutionQuerier {
	return &PostgresExecutionQuerier{}
}

// Pausing yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be paused by the runner.
func (a PostgresExecutionQuerier) Pausing(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	panic("not implemented") // TODO
}

// Resuming yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be resumed by the runner.
func (a PostgresExecutionQuerier) Resuming(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	panic("not implemented") // TODO
}

// Aborting yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be aborted by the runner.
func (a PostgresExecutionQuerier) Aborting(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	panic("not implemented") // TODO
}

// Cancelling yields an iterator returning all executions assigned to the runner indicated
// by the passed runner, that should be cancelled by the runner.
func (a PostgresExecutionQuerier) Cancelling(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool) {
	panic("not implemented") // TODO
}
