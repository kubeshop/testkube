package runner

import (
	"context"

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

type lazyRunner struct {
	accessor *Runner
}

func Lazy(accessor *Runner) Runner {
	return &lazyRunner{accessor: accessor}
}

func (r *lazyRunner) Monitor(ctx context.Context, organizationId, environmentId, id string) error {
	return (*r.accessor).Monitor(ctx, organizationId, environmentId, id)
}

func (r *lazyRunner) Notifications(ctx context.Context, id string) executionworkertypes.NotificationsWatcher {
	return (*r.accessor).Notifications(ctx, id)
}

func (r *lazyRunner) Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	return (*r.accessor).Execute(request)
}

func (r *lazyRunner) Pause(id string) error {
	return (*r.accessor).Pause(id)
}

func (r *lazyRunner) Resume(id string) error {
	return (*r.accessor).Resume(id)
}

func (r *lazyRunner) Abort(id string) error {
	return (*r.accessor).Abort(id)
}

type lazyRunnerExecute struct {
	accessor *RunnerExecute
}

func LazyExecute(accessor *RunnerExecute) RunnerExecute {
	return &lazyRunnerExecute{accessor: accessor}
}

func (r *lazyRunnerExecute) Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	return (*r.accessor).Execute(request)
}
