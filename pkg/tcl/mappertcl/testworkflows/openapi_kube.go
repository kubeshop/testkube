// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflows

import (
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapTestWorkflowRunningContextInterfaceTypeAPIToKube(v testkube.TestWorkflowRunningContextInterfaceType) testworkflowsv1.TestWorkflowRunningContextInterfaceType {
	return testworkflowsv1.TestWorkflowRunningContextInterfaceType(v)
}

func MapTestWorkflowRunningContextInterfaceAPIToKube(v testkube.TestWorkflowRunningContextInterface) testworkflowsv1.TestWorkflowRunningContextInterface {
	return testworkflowsv1.TestWorkflowRunningContextInterface{
		Name:  v.Name,
		Type_: common.MapPtr(v.Type_, MapTestWorkflowRunningContextInterfaceTypeAPIToKube),
	}
}

func MapTestWorkflowRunningContextActorTypeAPIToKube(v testkube.TestWorkflowRunningContextActorType) testworkflowsv1.TestWorkflowRunningContextActorType {
	return testworkflowsv1.TestWorkflowRunningContextActorType(v)
}

func MapTestWorkflowRunningContextActorAPIToKube(v testkube.TestWorkflowRunningContextActor) testworkflowsv1.TestWorkflowRunningContextActor {
	return testworkflowsv1.TestWorkflowRunningContextActor{
		Name:               v.Name,
		Email:              v.Email,
		ExecutionId:        v.ExecutionId,
		ExecutionPath:      v.ExecutionPath,
		ExecutionReference: v.ExecutionReference,
		Type_:              common.MapPtr(v.Type_, MapTestWorkflowRunningContextActorTypeAPIToKube),
	}
}

func MapTestWorkflowRunningContextAPIToKube(v testkube.TestWorkflowRunningContext) testworkflowsv1.TestWorkflowRunningContext {
	return testworkflowsv1.TestWorkflowRunningContext{
		Interface_: common.MapPtr(v.Interface_, MapTestWorkflowRunningContextInterfaceAPIToKube),
		Actor:      common.MapPtr(v.Actor, MapTestWorkflowRunningContextActorAPIToKube),
	}
}
