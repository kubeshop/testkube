// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package mapper

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

// MapTestkubeTestWorkflowRunningContextActorToCDEventTiggerType maps OpenAPI spec Test Workflow Running Context Actor to CDEvent Trigger Type
func MapTestkubeTestWorkflowRunningContextActorToCDEventTiggerType(actor testkube.TestWorkflowRunningContextActorType) string {
	switch actor {
	case testkube.USER_TestWorkflowRunningContextActorType, testkube.PROGRAM_TestWorkflowRunningContextActorType:
		return "manual"
	case testkube.TESTWORKFLOW_TestWorkflowRunningContextActorType, testkube.TESTWORKFLOWEXECUTION_TestWorkflowRunningContextActorType,
		testkube.TESTTRIGGER_TestWorkflowRunningContextActorType:
		return "event"
	case testkube.CRON_TestWorkflowRunningContextActorType:
		return "schedule"
	}

	return "other"
}
