// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt
package api

import (
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func GetRunningContext(name, username, email string) *testkube.TestWorkflowRunningContext {
	return &testkube.TestWorkflowRunningContext{
		Interface_: &testkube.TestWorkflowRunningContextInterface{
			Type_: common.Ptr(testkube.API_TestWorkflowRunningContextInterfaceType),
		},
		Actor: &testkube.TestWorkflowRunningContextActor{
			Name:     name,
			Username: username,
			Email:    email,
			Type_:    common.Ptr(testkube.PROGRAM_TestWorkflowRunningContextActorType),
		},
	}
}
