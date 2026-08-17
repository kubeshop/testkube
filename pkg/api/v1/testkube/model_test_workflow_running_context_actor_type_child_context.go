package testkube

import "github.com/kubeshop/testkube/pkg/cloud"

func (t TestWorkflowRunningContextActorType) ChildRunningContextType() cloud.RunningContextType {
	switch t {
	case QUALITYLOOP_TestWorkflowRunningContextActorType:
		return cloud.RunningContextType_QUALITYLOOP
	}
	return cloud.RunningContextType_EXECUTION
}
