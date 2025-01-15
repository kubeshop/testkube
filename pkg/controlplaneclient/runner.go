package controlplaneclient

import (
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

type RunnerClient interface {
	GetRunnerOngoingExecutions(ctx context.Context) ([]*cloud.UnfinishedExecution, error)
	WatchRunnerRequests(ctx context.Context) channels.Watcher[*cloud.RunnerRequest]
	ProcessExecutionNotificationRequests(ctx context.Context, process func(ctx context.Context, req *cloud.TestWorkflowNotificationsRequest) channels.Watcher[*testkube.TestWorkflowExecutionNotification]) error
	ProcessExecutionParallelWorkerNotificationRequests(ctx context.Context, process func(ctx context.Context, req *cloud.TestWorkflowParallelStepNotificationsRequest) channels.Watcher[*testkube.TestWorkflowExecutionNotification]) error
	ProcessExecutionServiceNotificationRequests(ctx context.Context, process func(ctx context.Context, req *cloud.TestWorkflowServiceNotificationsRequest) channels.Watcher[*testkube.TestWorkflowExecutionNotification]) error
}

func (c *client) GetRunnerOngoingExecutions(ctx context.Context) ([]*cloud.UnfinishedExecution, error) {
	if c.IsLegacy() {
		return c.legacyGetRunnerOngoingExecutions(ctx)
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
func (c *client) legacyGetRunnerOngoingExecutions(ctx context.Context) ([]*cloud.UnfinishedExecution, error) {
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

func (c *client) WatchRunnerRequests(ctx context.Context) channels.Watcher[*cloud.RunnerRequest] {
	stream, err := watch(ctx, c.metadata().GRPC(), c.client.GetRunnerRequests)
	if err != nil {
		return channels.NewError[*cloud.RunnerRequest](err)
	}
	watcher := channels.NewWatcher[*cloud.RunnerRequest]()
	go func() {
		defer watcher.Close(err)
		for {
			// Ignore if it's not implemented in the Control Plane
			if getGrpcErrorCode(err) == codes.Unimplemented {
				return
			}

			// Take the context error if possible
			if errors.Is(err, context.Canceled) && ctx.Err() != nil {
				err = context.Cause(ctx)
			}

			// Handle the error
			if err != nil {
				return
			}

			// Get the next runner request
			var req *cloud.RunnerRequestData
			req, err = stream.Recv()
			if err != nil {
				continue
			}

			if req.Ping {
				err = stream.Send(&cloud.RunnerResponseData{Ping: true})
				if err != nil {
					return
				}
				continue
			}

			watcher.Send(req.Request)
		}
	}()
	return watcher
}

func (c *client) ProcessExecutionNotificationRequests(ctx context.Context, process func(ctx context.Context, req *cloud.TestWorkflowNotificationsRequest) channels.Watcher[*testkube.TestWorkflowExecutionNotification]) error {
	return processNotifications(
		ctx,
		c.metadata().GRPC(),
		c.client.GetTestWorkflowNotificationsStream,
		buildPongNotification,
		buildCloudNotification,
		buildCloudError,
		process,
	)
}

func (c *client) ProcessExecutionParallelWorkerNotificationRequests(ctx context.Context, process func(ctx context.Context, req *cloud.TestWorkflowParallelStepNotificationsRequest) channels.Watcher[*testkube.TestWorkflowExecutionNotification]) error {
	return processNotifications(
		ctx,
		c.metadata().GRPC(),
		c.client.GetTestWorkflowParallelStepNotificationsStream,
		buildParallelStepPongNotification,
		buildParallelStepCloudNotification,
		buildParallelStepCloudError,
		process,
	)
}

func (c *client) ProcessExecutionServiceNotificationRequests(ctx context.Context, process func(ctx context.Context, req *cloud.TestWorkflowServiceNotificationsRequest) channels.Watcher[*testkube.TestWorkflowExecutionNotification]) error {
	return processNotifications(
		ctx,
		c.metadata().GRPC(),
		c.client.GetTestWorkflowServiceNotificationsStream,
		buildServicePongNotification,
		buildServiceCloudNotification,
		buildServiceCloudError,
		process,
	)
}
