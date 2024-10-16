package mapper

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

// MapTestkubeTestWorkflowRunningContextActorToCDEventTiggerType maps OpenAPI spec Test Workflow Running Context Actor to CDEvent Trigger Type
func MapTestkubeTestWorkflowRunningContextActorToCDEventTiggerType(actor testkube.TestWorkflowRunningContextActorType) string {
	switch actor {
	case testkube.USER_TestWorkflowRunningContextActorType, testkube.PROGRAM_TestWorkflowRunningContextActorType:
		return "manual"
	case testkube.TESTWORKFLOW_TestWorkflowRunningContextActorType, testkube.TESTWORKFLOWEXECUTION_TestWorkflowRunningContextActorType,
		testkube.TESTRIGGER_TestWorkflowRunningContextActorType:
		return "event"
	case testkube.CRON_TestWorkflowRunningContextActorType:
		return "schedule"
	}

	return "other"
}
