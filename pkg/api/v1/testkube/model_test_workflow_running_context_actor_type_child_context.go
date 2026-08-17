package testkube

import "github.com/kubeshop/testkube/pkg/cloud"

// ChildRunningContextType returns the cloud RunningContextType that a chained
// child scheduled from a parent with this actor type should carry.
//
// The default for every actor is RunningContextType_EXECUTION, which the
// server maps back to actor.type = testworkflow (the "Workflow" chip on the
// Executions page). Only actors listed in the switch overwrite the default,
// causing children to inherit the parent's actor natively. Extend the switch
// carefully — every entry added here changes the wire semantics of every
// chained child scheduled by that actor's parents.
func (t TestWorkflowRunningContextActorType) ChildRunningContextType() cloud.RunningContextType {
	switch t {
	case QUALITYLOOP_TestWorkflowRunningContextActorType:
		return cloud.RunningContextType_QUALITYLOOP
	}
	return cloud.RunningContextType_EXECUTION
}
