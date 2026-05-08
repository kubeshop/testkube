// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

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
			Type_: common.Ptr(testkube.TESTTRIGGER_TestWorkflowRunningContextActorType),
		},
	}
}
