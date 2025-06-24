// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package execute

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	commonmapper "github.com/kubeshop/testkube/pkg/mapper/common"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
)

func ExecuteTestWorkflow(workflowName string, request testkube.TestWorkflowExecutionRequest) ([]testkube.TestWorkflowExecution, error) {
	if env.IsNewArchitecture() {
		return executeTestWorkflowGrpc(workflowName, request)
	}
	return executeTestWorkflowApi(workflowName, request)
}

func executeTestWorkflowApi(workflowName string, request testkube.TestWorkflowExecutionRequest) ([]testkube.TestWorkflowExecution, error) {
	cfg := config.Config()
	client := env.Testkube()

	parentIds := make([]string, 0)
	if cfg.Execution.ParentIds != "" {
		parentIds = strings.Split(cfg.Execution.ParentIds, "/")
	}
	parentIds = append(parentIds, cfg.Execution.Id)

	request.ParentExecutionIds = parentIds
	request.RunningContext = &testkube.TestWorkflowRunningContext{
		Interface_: &testkube.TestWorkflowRunningContextInterface{
			Type_: common.Ptr(testkube.API_TestWorkflowRunningContextInterfaceType),
		},
		Actor: &testkube.TestWorkflowRunningContextActor{
			Name:          cfg.Workflow.Name,
			ExecutionId:   cfg.Execution.Id,
			ExecutionPath: strings.Join(parentIds, "/"),
			Type_:         common.Ptr(testkube.TESTWORKFLOW_TestWorkflowRunningContextActorType),
		},
	}

	execution, err := client.ExecuteTestWorkflow(workflowName, request)
	if err != nil {
		return nil, err
	}
	return []testkube.TestWorkflowExecution{execution}, nil
}

func executeTestWorkflowGrpc(workflowName string, request testkube.TestWorkflowExecutionRequest) ([]testkube.TestWorkflowExecution, error) {
	cfg := config.Config()
	client, err := env.Cloud()
	if err != nil {
		return nil, err
	}
	var targets []*cloud.ExecutionTarget
	if request.Target != nil {
		targets = commonmapper.MapAllTargetsApiToGrpc([]testkube.ExecutionTarget{*request.Target})
	}

	return client.ScheduleExecution(context.Background(), cfg.Execution.EnvironmentId, &cloud.ScheduleRequest{
		Executions:      []*cloud.ScheduleExecution{{Selector: &cloud.ScheduleResourceSelector{Name: workflowName}, Config: request.Config, Targets: targets}},
		DisableWebhooks: cfg.Execution.DisableWebhooks,
		Tags:            request.Tags,
	}).All()
}

func ListTestWorkflows(labels map[string]string) ([]testkube.TestWorkflow, error) {
	if env.IsExternalStorage() {
		return listTestWorkflowsGrpc(labels)
	}
	return listTestWorkflowsApi(labels)
}

func listTestWorkflowsApi(labels map[string]string) ([]testkube.TestWorkflow, error) {
	client := env.Testkube()
	selectors := make([]string, 0, len(labels))
	for k, v := range labels {
		selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
	}
	return client.ListTestWorkflows(strings.Join(selectors, ","))
}

func listTestWorkflowsGrpc(labels map[string]string) ([]testkube.TestWorkflow, error) {
	cfg := config.Config()
	cloud, err := env.Cloud()
	if err != nil {
		return nil, err
	}
	client := testworkflowclient.NewCloudTestWorkflowClient(cloud)
	return client.List(context.Background(), cfg.Execution.EnvironmentId, testworkflowclient.ListOptions{Labels: labels})
}

func GetExecution(id string) (*testkube.TestWorkflowExecution, error) {
	cfg := config.Config()
	cloud, err := env.Cloud()
	if err != nil {
		return nil, err
	}
	return cloud.GetExecution(context.Background(), cfg.Execution.EnvironmentId, id)
}
