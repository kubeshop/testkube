package scheduling

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type ExecutionQuerier interface {
	ByStatus(ctx context.Context, statuses []testkube.TestWorkflowStatus) func(yield func(testkube.TestWorkflowExecution, error) bool)

	Pausing(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool)
	Resuming(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool)
	Aborting(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool)
	Cancelling(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool)

	Starting(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool)
	Assigned(ctx context.Context) func(yield func(testkube.TestWorkflowExecution, error) bool)
}
