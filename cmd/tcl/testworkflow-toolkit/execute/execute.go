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
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
)

func ExecuteTestWorkflow(workflowName string, request testkube.TestWorkflowExecutionRequest) ([]testkube.TestWorkflowExecution, error) {
	if env.IsNewExecutions() {
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
	md := metadata.New(map[string]string{"api-key": cfg.Worker.Connection.ApiKey, "organization-id": cfg.Execution.OrganizationId, "agent-id": cfg.Worker.Connection.AgentID})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, client := env.Cloud(ctx)

	parentIds := make([]string, 0)
	if cfg.Execution.ParentIds != "" {
		parentIds = strings.Split(cfg.Execution.ParentIds, "/")
	}
	parentIds = append(parentIds, cfg.Execution.Id)

	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	resp, err := client.ScheduleExecution(ctx, &cloud.ScheduleRequest{
		EnvironmentId:   cfg.Execution.EnvironmentId,
		Executions:      []*cloud.ScheduleExecution{{Selector: &cloud.ScheduleResourceSelector{Name: workflowName}, Config: request.Config}},
		DisableWebhooks: cfg.Execution.DisableWebhooks,
		Tags:            request.Tags,
		RunningContext: &cloud.RunningContext{
			Name: cfg.Execution.Id,
			Type: cloud.RunningContextType_EXECUTION,
		},
		ParentExecutionIds: parentIds,
	}, opts...)

	if err != nil {
		return nil, err
	}

	result := make([]testkube.TestWorkflowExecution, 0)
	var item *cloud.ScheduleResponse
	for {
		item, err = resp.Recv()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Printf("warn: %s\n", err)
			}
			break
		}
		var execution testkube.TestWorkflowExecution
		err = json.Unmarshal(item.Execution, &execution)
		if err != nil {
			fmt.Printf("warn: %s\n", err)
			break
		}
		result = append(result, execution)
	}

	return result, nil
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
	_, grpcClient := env.Cloud(context.Background())
	client := testworkflowclient.NewCloudTestWorkflowClient(grpcClient, cfg.Worker.Connection.ApiKey)
	return client.List(context.Background(), cfg.Execution.EnvironmentId, testworkflowclient.ListOptions{Labels: labels})
}

func GetExecution(id string) (*testkube.TestWorkflowExecution, error) {
	if env.IsNewExecutions() {
		return getExecution(id)
	}
	return getExecutionLegacy(id)
}

func getExecution(id string) (*testkube.TestWorkflowExecution, error) {
	cfg := config.Config()
	md := metadata.New(map[string]string{"api-key": cfg.Worker.Connection.ApiKey, "organization-id": cfg.Execution.OrganizationId, "agent-id": cfg.Worker.Connection.AgentID})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, client := env.Cloud(ctx)

	parentIds := make([]string, 0)
	if cfg.Execution.ParentIds != "" {
		parentIds = strings.Split(cfg.Execution.ParentIds, "/")
	}
	parentIds = append(parentIds, cfg.Execution.Id)

	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	resp, err := client.GetExecution(ctx, &cloud.GetExecutionRequest{EnvironmentId: cfg.Execution.EnvironmentId, Id: id}, opts...)
	if err != nil {
		return nil, err
	}
	var v testkube.TestWorkflowExecution
	err = json.Unmarshal(resp.Execution, &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func getExecutionLegacy(id string) (*testkube.TestWorkflowExecution, error) {
	c, _ := env.Cloud(context.Background())
	resp, err := c.Execute(context.Background(), testworkflow.CmdTestWorkflowExecutionGet, testworkflow.ExecutionGetRequest{ID: id})
	if err != nil {
		return nil, err
	}
	var v testworkflow.ExecutionGetResponse
	err = json.Unmarshal(resp, &v)
	if err != nil {
		return nil, err
	}
	return &v.WorkflowExecution, nil
}
