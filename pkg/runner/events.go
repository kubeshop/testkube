package runner

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
)

func notifyWorkflowCompleted(emitter event.Interface, execution *testkube.TestWorkflowExecution) {
	if emitter == nil || execution == nil || execution.Result == nil {
		return
	}

	groupId := execution.GroupId

	switch {
	case execution.Result.IsPassed():
		emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(execution, groupId))
	case execution.Result.IsAborted():
		emitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution, groupId))
	case execution.Result.IsCanceled():
		emitter.Notify(testkube.NewEventEndTestWorkflowCanceled(execution, groupId))
	default:
		emitter.Notify(testkube.NewEventEndTestWorkflowFailed(execution, groupId))
	}

	if execution.Result.IsNotPassed() {
		emitter.Notify(testkube.NewEventEndTestWorkflowNotPassed(execution, groupId))
	}
}
