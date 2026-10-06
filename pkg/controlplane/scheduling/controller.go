package scheduling

import "context"

type Controller interface {
	// StartExecutions moves a dispatched batch from ASSIGNED to STARTING and
	// returns the ids it actually claimed.
	//
	// The return value is load-bearing: an execution aborted between the dispatch
	// read and this write is skipped by the guarded update, and the caller must
	// dispatch only what was claimed or the runner creates Kubernetes resources
	// for an execution that is already over.
	//
	// Batched because the dispatch handler claims a whole page at once: doing it
	// per row cost two round trips per execution on every poll, inside the
	// request the runner is waiting on.
	StartExecutions(ctx context.Context, executionIds []string) ([]string, error)

	// RefreshStartingExecutions renews the dispatch lease on rows that were
	// re-offered to the runner, so a row being retried is not offered again on
	// the next poll. It returns the ids it renewed, for the same reason
	// StartExecutions does.
	RefreshStartingExecutions(ctx context.Context, executionIds []string) ([]string, error)

	PauseExecution(ctx context.Context, executionId string) error
	ResumeExecution(ctx context.Context, executionId string) error
	AbortExecution(ctx context.Context, executionId string) error
	CancelExecution(ctx context.Context, executionId string) error
	ForceCancelExecution(ctx context.Context, executionId string) error
}
