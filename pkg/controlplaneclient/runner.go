package controlplaneclient

import (
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeshop/testkube/pkg/cloud"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
)

type RunnerClient interface {
	GetUnfinishedExecutions(ctx context.Context) ([]*cloud.UnfinishedExecution, error)
}

func (c *client) GetUnfinishedExecutions(ctx context.Context) ([]*cloud.UnfinishedExecution, error) {
	if c.IsLegacy() {
		return c.legacyGetUnfinishedExecutions(ctx)
	}
	res, err := call(ctx, c.metadata().GRPC(), c.client.GetUnfinishedExecutions, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	result := make([]*cloud.UnfinishedExecution, 0)
	for {
		// Take the context error if possible
		if err == nil && ctx.Err() != nil {
			err = ctx.Err()
		}

		// End when it's done
		if errors.Is(err, io.EOF) {
			return result, nil
		}

		// Handle the error
		if err != nil {
			return nil, err
		}

		// Get the next execution to monitor
		var exec *cloud.UnfinishedExecution
		exec, err = res.Recv()
		if err != nil {
			continue
		}

		result = append(result, exec)
	}
}

// Deprecated
func (c *client) legacyGetUnfinishedExecutions(ctx context.Context) ([]*cloud.UnfinishedExecution, error) {
	jsonPayload, err := json.Marshal(cloudtestworkflow.ExecutionGetRunningRequest{})
	if err != nil {
		return nil, err
	}
	s := structpb.Struct{}
	if err := s.UnmarshalJSON(jsonPayload); err != nil {
		return nil, err
	}
	req := cloud.CommandRequest{
		Command: string(cloudtestworkflow.CmdTestWorkflowExecutionGetRunning),
		Payload: &s,
	}
	cmdResponse, err := call(ctx, c.metadata().GRPC(), c.client.Call, &req)
	if err != nil {
		return nil, err
	}
	var response cloudtestworkflow.ExecutionGetRunningResponse
	err = json.Unmarshal(cmdResponse.Response, &response)
	if err != nil {
		return nil, err
	}
	result := make([]*cloud.UnfinishedExecution, 0)
	for i := range response.WorkflowExecutions {
		// Ignore if it's not assigned to any runner
		if response.WorkflowExecutions[i].RunnerId == "" && len(response.WorkflowExecutions[i].Signature) == 0 {
			continue
		}

		// Ignore if it's assigned to a different runner
		if response.WorkflowExecutions[i].RunnerId != c.agentID {
			continue
		}

		result = append(result, &cloud.UnfinishedExecution{EnvironmentId: c.proContext.EnvID, Id: response.WorkflowExecutions[i].Id})
	}
	return result, err
}
