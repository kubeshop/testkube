package mapper

import (
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func NewMockRunningContext() *testkube.TestWorkflowRunningContext {
	return &testkube.TestWorkflowRunningContext{
		Actor: &testkube.TestWorkflowRunningContextActor{
			Type_: common.Ptr(testkube.CRON_TestWorkflowRunningContextActorType),
		},
	}
}
