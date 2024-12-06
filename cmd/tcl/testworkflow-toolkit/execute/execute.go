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
)

var (
	isGrpcExecuteMu sync.Mutex
	isGrpcCache     *bool
)

func isGrpcExecute() bool {
	isGrpcExecuteMu.Lock()
	defer isGrpcExecuteMu.Unlock()

	if isGrpcCache == nil {
		cfg := config.Config()
		ctx := agentclient.AddAPIKeyMeta(context.Background(), cfg.Worker.Connection.ApiKey)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, client := env.Cloud(ctx)
		proContext, _ := client.GetProContext(ctx, &emptypb.Empty{})
		if proContext != nil {
			isGrpcCache = common.Ptr(capabilities.Enabled(proContext.Capabilities, capabilities.CapabilityNewExecutions))
		} else {
			isGrpcCache = common.Ptr(false)
		}
	}
	return *isGrpcCache
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
		Selectors:       []*cloud.ScheduleSelector{{Name: workflowName, Config: request.Config}},
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
