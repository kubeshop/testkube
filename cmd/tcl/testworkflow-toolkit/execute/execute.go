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
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/internal/common"
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

var (
	isGrpcMu           sync.Mutex
	isGrpcExecuteCache *bool
	isGrpcListCache    *bool
)

func loadCapabilities() {
	isGrpcMu.Lock()
	defer isGrpcMu.Unlock()

	// Block if the instance doesn't support that
	cfg := config.Config()
	if isGrpcExecuteCache == nil && cfg.Worker.FeatureFlags[testworkflowconfig.FeatureFlagNewExecutions] != "true" {
		isGrpcExecuteCache = common.Ptr(false)
	}
	if isGrpcListCache == nil && cfg.Worker.FeatureFlags[testworkflowconfig.FeatureFlagTestWorkflowCloudStorage] != "true" {
		isGrpcListCache = common.Ptr(false)
	}

	// Do not check Cloud support if its already predefined
	if isGrpcExecuteCache != nil && isGrpcListCache != nil {
		return
	}

	// Check support in the cloud
	ctx := agentclient.AddAPIKeyMeta(context.Background(), cfg.Worker.Connection.ApiKey)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, client := env.Cloud(ctx)
	proContext, _ := client.GetProContext(ctx, &emptypb.Empty{})
	if proContext != nil {
		if isGrpcExecuteCache == nil {
			isGrpcExecuteCache = common.Ptr(capabilities.Enabled(proContext.Capabilities, capabilities.CapabilityNewExecutions))
		}
		if isGrpcListCache == nil {
			isGrpcListCache = common.Ptr(capabilities.Enabled(proContext.Capabilities, capabilities.CapabilityTestWorkflowStorage))
		}
	} else {
		isGrpcExecuteCache = common.Ptr(false)
		isGrpcListCache = common.Ptr(false)
	}
}

func isGrpcExecute() bool {
	loadCapabilities()
	return *isGrpcExecuteCache
}

func isGrpcList() bool {
	loadCapabilities()
	return *isGrpcListCache
}

func ExecuteTestWorkflow(workflowName string, request testkube.TestWorkflowExecutionRequest) ([]testkube.TestWorkflowExecution, error) {
	if isGrpcExecute() {
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
	ctx := agentclient.AddAPIKeyMeta(context.Background(), cfg.Worker.Connection.ApiKey)
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
	if isGrpcList() {
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
