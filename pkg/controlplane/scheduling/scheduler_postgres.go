package scheduling

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type PostgresScheduler struct{}

func NewPostgresScheduler() Scheduler {
	return &PostgresScheduler{}
}

func (s *PostgresScheduler) ScheduleExecution(ctx context.Context, info RunnerInfo) (execution testkube.TestWorkflowExecution, found bool, e error) {
	// Note: Standalone Control Plane does not support policies.
	// Note: Standalone Control Plane does not support label matches, excludes, etc. It always targets the DefaultRunner.
	panic("implement me") // TODO
}
