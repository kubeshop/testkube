package scheduling

import (
	"context"
)

type PostgresExecutionController struct{}

func NewPostgresExecutionController() *PostgresExecutionController {
	return &PostgresExecutionController{}
}

// StartExecution marks an execution that is currently assigned that it should be started.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, then no error will be emitted and no action will have been taken.
func (a PostgresExecutionController) StartExecution(ctx context.Context, executionId string) error {
	panic("implement me") // TODO
}

// PauseExecution marks an execution that is currently running that it should be paused.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is currently running, then no error will be emitted and no action will
// have been taken.
func (a PostgresExecutionController) PauseExecution(ctx context.Context, executionId string) error {
	panic("implement me") // TODO
}

// ResumeExecution marks an execution that is currently paused that it should be resumed.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is currently paused, then no error will be emitted and no action will
// have been taken.
func (a PostgresExecutionController) ResumeExecution(ctx context.Context, executionId string) error {
	panic("implement me") // TODO
}

// AbortExecution marks an execution that is currently in an executing state that it
// should be aborted.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is in an appropriate state, then no error will be emitted and no action will
// have been taken.
// Executions can only be aborted if they are currently in a Starting, Running, Paused, or
// Resuming state.
func (a PostgresExecutionController) AbortExecution(ctx context.Context, executionId string) error {
	panic("implement me") // TODO
}

// CancelExecution marks an execution that is currently in an executing state that it
// should be cancelled.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is in an appropriate state, then no error will be emitted and no action will
// have been taken.
// Executions can only be cancelled if they are currently in a Starting, Running, Paused, or
// Resuming state.
func (a PostgresExecutionController) CancelExecution(ctx context.Context, executionId string) error {
	panic("implement me") // TODO
}

// ForceCancelExecution cancels an execution and all its steps
// This mirrors the MongoDB implementation's behavior:
// 1. Updates the main execution status_at timestamp
// 2. Sets the result status to 'canceled'
// 3. Cancels all non-terminated steps (preserves passed/failed steps)
// 4. Sets missing timestamps (queuedat, startedat, finishedat) to current time
// 5. Cancels the initialization step if present
func (a *PostgresExecutionController) ForceCancelExecution(ctx context.Context, executionId string) error {
	panic("implement me") // TODO
}
