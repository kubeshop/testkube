package triggers

import (
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func GetRunningContext(name string) *testkube.TestWorkflowRunningContext {
	return &testkube.TestWorkflowRunningContext{
		Interface_: &testkube.TestWorkflowRunningContextInterface{
			Type_: common.Ptr(testkube.INTERNAL_TestWorkflowRunningContextInterfaceType),
		},
		Actor: &testkube.TestWorkflowRunningContextActor{
			Name:  name,
			Type_: common.Ptr(testkube.TESTRIGGER_TestWorkflowRunningContextActorType),
		},
	}
}
