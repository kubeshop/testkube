package scheduling

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type Scheduler interface {
	ScheduleExecution(ctx context.Context, info RunnerInfo) (execution testkube.TestWorkflowExecution, found bool, err error)
}

type RunnerInfo struct {
	Id            string
	Name          string
	EnvironmentId string
}
