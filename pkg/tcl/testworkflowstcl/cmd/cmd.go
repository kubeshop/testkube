// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package cmd

import (
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func GetRunningContext(runContext, username, email string, interfaceType testkube.TestWorkflowRunningContextInterfaceType) *testkube.TestWorkflowRunningContext {
	return &testkube.TestWorkflowRunningContext{
		Interface_: &testkube.TestWorkflowRunningContextInterface{
			Name:  runContext,
			Type_: common.Ptr(interfaceType),
		},
		Actor: &testkube.TestWorkflowRunningContextActor{
			Type_:    common.Ptr(testkube.USER_TestWorkflowRunningContextActorType),
			Username: username,
			Email:    email,
		},
	}
}
