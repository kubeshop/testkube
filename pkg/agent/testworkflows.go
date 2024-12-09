package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"

	"github.com/kubeshop/testkube/internal/common"
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
)

const testWorkflowNotificationsRetryCount = 10

var (
	logRetryDelay = 100 * time.Millisecond
)

func getTestWorkflowNotificationType(n testkube.TestWorkflowExecutionNotification) cloud.TestWorkflowNotificationType {
	if n.Result != nil {
		return cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_RESULT
	} else if n.Output != nil {
		return cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_OUTPUT
	}
	return cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_LOG
}

func (ag *Agent) runTestWorkflowNotificationsLoop(ctx context.Context) error {
	ctx = agentclient.AddAPIKeyMeta(ctx, ag.apiKey)

	ag.logger.Infow("initiating workflow notifications streaming connection with Cloud API")
	// creates a new Stream from the client side. ctx is used for the lifetime of the stream.
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	stream, err := ag.client.GetTestWorkflowNotificationsStream(ctx, opts...)
	if err != nil {
		ag.logger.Errorf("failed to execute: %w", err)
		return errors.Wrap(err, "failed to setup stream")
	}

	// GRPC stream have special requirements for concurrency on SendMsg, and RecvMsg calls.
	// Please check https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			cmd, err := ag.receiveTestWorkflowNotificationsRequest(groupCtx, stream)
			if err != nil {
				return err
			}

			ag.testWorkflowNotificationsRequestBuffer <- cmd
		}
	})

	g.Go(func() error {
		for {
			select {
			case resp := <-ag.testWorkflowNotificationsResponseBuffer:
				err := ag.sendTestWorkflowNotificationsResponse(groupCtx, stream, resp)
				if err != nil {
					return err
				}
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
	})

	err = g.Wait()

	return err
}

func (ag *Agent) runTestWorkflowServiceNotificationsLoop(ctx context.Context) error {
	ctx = agentclient.AddAPIKeyMeta(ctx, ag.apiKey)

	ag.logger.Infow("initiating workflow service notifications streaming connection with Cloud API")
	// creates a new Stream from the client side. ctx is used for the lifetime of the stream.
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	stream, err := ag.client.GetTestWorkflowServiceNotificationsStream(ctx, opts...)
	if err != nil {
		ag.logger.Errorf("failed to execute: %w", err)
		return errors.Wrap(err, "failed to setup stream")
	}

	// GRPC stream have special requirements for concurrency on SendMsg, and RecvMsg calls.
	// Please check https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			cmd, err := ag.receiveTestWorkflowServiceNotificationsRequest(groupCtx, stream)
			if err != nil {
				return err
			}

			ag.testWorkflowServiceNotificationsRequestBuffer <- cmd
		}
	})

	g.Go(func() error {
		for {
			select {
			case resp := <-ag.testWorkflowServiceNotificationsResponseBuffer:
				err := ag.sendTestWorkflowServiceNotificationsResponse(groupCtx, stream, resp)
				if err != nil {
					return err
				}
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
	})

	err = g.Wait()

	return err
}

func (ag *Agent) runTestWorkflowParallelStepNotificationsLoop(ctx context.Context) error {
	ctx = agentclient.AddAPIKeyMeta(ctx, ag.apiKey)

	ag.logger.Infow("initiating workflow parallel step notifications streaming connection with Cloud API")
	// creates a new Stream from the client side. ctx is used for the lifetime of the stream.
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	stream, err := ag.client.GetTestWorkflowParallelStepNotificationsStream(ctx, opts...)
	if err != nil {
		ag.logger.Errorf("failed to execute: %w", err)
		return errors.Wrap(err, "failed to setup stream")
	}

	// GRPC stream have special requirements for concurrency on SendMsg, and RecvMsg calls.
	// Please check https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			cmd, err := ag.receiveTestWorkflowParallelStepNotificationsRequest(groupCtx, stream)
			if err != nil {
				return err
			}

			ag.testWorkflowParallelStepNotificationsRequestBuffer <- cmd
		}
	})

	g.Go(func() error {
		for {
			select {
			case resp := <-ag.testWorkflowParallelStepNotificationsResponseBuffer:
				err := ag.sendTestWorkflowParallelStepNotificationsResponse(groupCtx, stream, resp)
				if err != nil {
					return err
				}
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
	})

	err = g.Wait()

	return err
}

func (ag *Agent) runTestWorkflowNotificationsWorker(ctx context.Context, numWorkers int) error {
	g, groupCtx := errgroup.WithContext(ctx)
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for {
				select {
				case req := <-ag.testWorkflowNotificationsRequestBuffer:
					if req.RequestType == cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK {
						ag.testWorkflowNotificationsResponseBuffer <- &cloud.TestWorkflowNotificationsResponse{
							StreamId: req.StreamId,
							SeqNo:    0,
						}
						break
					}

					err := ag.executeWorkflowNotificationsRequest(groupCtx, req)
					if err != nil {
						ag.logger.Errorf("error executing workflow notifications request: %s", err.Error())
					}
				case <-groupCtx.Done():
					return groupCtx.Err()
				}
			}
		})
	}
	return g.Wait()
}

func (ag *Agent) runTestWorkflowServiceNotificationsWorker(ctx context.Context, numWorkers int) error {
	g, groupCtx := errgroup.WithContext(ctx)
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for {
				select {
				case req := <-ag.testWorkflowServiceNotificationsRequestBuffer:
					if req.RequestType == cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK {
						ag.testWorkflowServiceNotificationsResponseBuffer <- &cloud.TestWorkflowServiceNotificationsResponse{
							StreamId: req.StreamId,
							SeqNo:    0,
						}
						break
					}

					err := ag.executeWorkflowServiceNotificationsRequest(groupCtx, req)
					if err != nil {
						ag.logger.Errorf("error executing workflow service notifications request: %s", err.Error())
					}
				case <-groupCtx.Done():
					return groupCtx.Err()
				}
			}
		})
	}
	return g.Wait()
}

func (ag *Agent) runTestWorkflowParallelStepNotificationsWorker(ctx context.Context, numWorkers int) error {
	g, groupCtx := errgroup.WithContext(ctx)
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for {
				select {
				case req := <-ag.testWorkflowParallelStepNotificationsRequestBuffer:
					if req.RequestType == cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK {
						ag.testWorkflowParallelStepNotificationsResponseBuffer <- &cloud.TestWorkflowParallelStepNotificationsResponse{
							StreamId: req.StreamId,
							SeqNo:    0,
						}
						break
					}

					err := ag.executeWorkflowParallelStepNotificationsRequest(groupCtx, req)
					if err != nil {
						ag.logger.Errorf("error executing workflow parallel step notifications request: %s", err.Error())
					}
				case <-groupCtx.Done():
					return groupCtx.Err()
				}
			}
		})
	}
	return g.Wait()
}

func (ag *Agent) executeWorkflowNotificationsRequest(ctx context.Context, req *cloud.TestWorkflowNotificationsRequest) error {
	notificationsCh, err := ag.testWorkflowNotificationsFunc(ctx, req.ExecutionId)
	for i := 0; i < testWorkflowNotificationsRetryCount; i++ {
		if err != nil {
			// We have a race condition here
			// Cloud sometimes slow to insert execution or test
			// while WorkflowNotifications request from websockets comes in faster
			// so we retry up to testWorkflowNotificationsRetryCount times.
			time.Sleep(logRetryDelay)
			notificationsCh, err = ag.testWorkflowNotificationsFunc(ctx, req.ExecutionId)
		}
	}
	if err != nil {
		message := fmt.Sprintf("cannot get pod logs: %s", err.Error())
		ag.testWorkflowNotificationsResponseBuffer <- &cloud.TestWorkflowNotificationsResponse{
			StreamId: req.StreamId,
			SeqNo:    0,
			Type:     cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_ERROR,
			Message:  fmt.Sprintf("%s %s", time.Now().Format(controller.KubernetesLogTimeFormat), message),
		}
		return nil
	}

	for {
		var i uint32
		select {
		case n, ok := <-notificationsCh:
			if !ok {
				return nil
			}
			t := getTestWorkflowNotificationType(n)
			msg := &cloud.TestWorkflowNotificationsResponse{
				StreamId:  req.StreamId,
				SeqNo:     i,
				Timestamp: n.Ts.Format(time.RFC3339Nano),
				Ref:       n.Ref,
				Type:      t,
			}
			if n.Result != nil {
				m, _ := json.Marshal(n.Result)
				msg.Message = string(m)
			} else if n.Output != nil {
				m, _ := json.Marshal(n.Output)
				msg.Message = string(m)
			} else {
				msg.Message = n.Log
			}
			i++

			select {
			case ag.testWorkflowNotificationsResponseBuffer <- msg:
			case <-ctx.Done():
				return ctx.Err()
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (ag *Agent) executeWorkflowServiceNotificationsRequest(ctx context.Context, req *cloud.TestWorkflowServiceNotificationsRequest) error {
	notificationsCh, err := retry.DoWithData(
		func() (<-chan testkube.TestWorkflowExecutionNotification, error) {
			// We have a race condition here
			// Cloud sometimes slow to start service
			// while WorkflowNotifications request from websockets comes in faster
			// so we retry up to wait till service pod is uo or execution is finished.
			return ag.testWorkflowServiceNotificationsFunc(ctx, req.ExecutionId, req.ServiceName, int(req.ServiceIndex))
		},
		retry.DelayType(retry.FixedDelay),
		retry.Delay(logRetryDelay),
		retry.RetryIf(func(err error) bool {
			return errors.Is(err, registry.ErrResourceNotFound)
		}),
		retry.UntilSucceeded(),
	)

	if err != nil {
		message := fmt.Sprintf("cannot get service pod logs: %s", err.Error())
		ag.testWorkflowServiceNotificationsResponseBuffer <- &cloud.TestWorkflowServiceNotificationsResponse{
			StreamId: req.StreamId,
			SeqNo:    0,
			Type:     cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_ERROR,
			Message:  fmt.Sprintf("%s %s", time.Now().Format(controller.KubernetesLogTimeFormat), message),
		}
		return nil
	}

	for {
		var i uint32
		select {
		case n, ok := <-notificationsCh:
			if !ok {
				return nil
			}
			t := getTestWorkflowNotificationType(n)
			msg := &cloud.TestWorkflowServiceNotificationsResponse{
				StreamId:  req.StreamId,
				SeqNo:     i,
				Timestamp: n.Ts.Format(time.RFC3339Nano),
				Ref:       n.Ref,
				Type:      t,
			}
			if n.Result != nil {
				m, _ := json.Marshal(n.Result)
				msg.Message = string(m)
			} else if n.Output != nil {
				m, _ := json.Marshal(n.Output)
				msg.Message = string(m)
			} else {
				msg.Message = n.Log
			}
			i++

			select {
			case ag.testWorkflowServiceNotificationsResponseBuffer <- msg:
			case <-ctx.Done():
				return ctx.Err()
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (ag *Agent) executeWorkflowParallelStepNotificationsRequest(ctx context.Context, req *cloud.TestWorkflowParallelStepNotificationsRequest) error {
	notificationsCh, err := retry.DoWithData(
		func() (<-chan testkube.TestWorkflowExecutionNotification, error) {
			// We have a race condition here
			// Cloud sometimes slow to start parallel step
			// while WorkflowNotifications request from websockets comes in faster
			// so we retry up to wait till parallel step pod is uo or execution is finished.
			return ag.testWorkflowParallelStepNotificationsFunc(ctx, req.ExecutionId, req.Ref, int(req.WorkerIndex))
		},
		retry.DelayType(retry.FixedDelay),
		retry.Delay(logRetryDelay),
		retry.RetryIf(func(err error) bool {
			return errors.Is(err, registry.ErrResourceNotFound)
		}),
		retry.UntilSucceeded(),
	)

	if err != nil {
		message := fmt.Sprintf("cannot get parallel step pod logs: %s", err.Error())
		ag.testWorkflowParallelStepNotificationsResponseBuffer <- &cloud.TestWorkflowParallelStepNotificationsResponse{
			StreamId: req.StreamId,
			SeqNo:    0,
			Type:     cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_ERROR,
			Message:  fmt.Sprintf("%s %s", time.Now().Format(controller.KubernetesLogTimeFormat), message),
		}
		return nil
	}

	for {
		var i uint32
		select {
		case n, ok := <-notificationsCh:
			if !ok {
				return nil
			}
			t := getTestWorkflowNotificationType(n)
			msg := &cloud.TestWorkflowParallelStepNotificationsResponse{
				StreamId:  req.StreamId,
				SeqNo:     i,
				Timestamp: n.Ts.Format(time.RFC3339Nano),
				Ref:       n.Ref,
				Type:      t,
			}
			if n.Result != nil {
				m, _ := json.Marshal(n.Result)
				msg.Message = string(m)
			} else if n.Output != nil {
				m, _ := json.Marshal(n.Output)
				msg.Message = string(m)
			} else {
				msg.Message = n.Log
			}
			i++

			select {
			case ag.testWorkflowParallelStepNotificationsResponseBuffer <- msg:
			case <-ctx.Done():
				return ctx.Err()
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (ag *Agent) receiveTestWorkflowNotificationsRequest(ctx context.Context, stream cloud.TestKubeCloudAPI_GetTestWorkflowNotificationsStreamClient) (*cloud.TestWorkflowNotificationsRequest, error) {
	respChan := make(chan testWorkflowNotificationsRequest, 1)
	go func() {
		cmd, err := stream.Recv()
		respChan <- testWorkflowNotificationsRequest{resp: cmd, err: err}
	}()

	var cmd *cloud.TestWorkflowNotificationsRequest
	select {
	case resp := <-respChan:
		cmd = resp.resp
		err := resp.err

		if err != nil {
			ag.logger.Errorf("agent stream receive: %v", err)
			return nil, err
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return cmd, nil
}

type testWorkflowNotificationsRequest struct {
	resp *cloud.TestWorkflowNotificationsRequest
	err  error
}

func (ag *Agent) receiveTestWorkflowServiceNotificationsRequest(ctx context.Context, stream cloud.TestKubeCloudAPI_GetTestWorkflowServiceNotificationsStreamClient) (*cloud.TestWorkflowServiceNotificationsRequest, error) {
	respChan := make(chan testWorkflowServiceNotificationsRequest, 1)
	go func() {
		cmd, err := stream.Recv()
		respChan <- testWorkflowServiceNotificationsRequest{resp: cmd, err: err}
	}()

	var cmd *cloud.TestWorkflowServiceNotificationsRequest
	select {
	case resp := <-respChan:
		cmd = resp.resp
		err := resp.err

		if err != nil {
			ag.logger.Errorf("agent stream receive: %v", err)
			return nil, err
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return cmd, nil
}

type testWorkflowServiceNotificationsRequest struct {
	resp *cloud.TestWorkflowServiceNotificationsRequest
	err  error
}

func (ag *Agent) receiveTestWorkflowParallelStepNotificationsRequest(ctx context.Context, stream cloud.TestKubeCloudAPI_GetTestWorkflowParallelStepNotificationsStreamClient) (*cloud.TestWorkflowParallelStepNotificationsRequest, error) {
	respChan := make(chan testWorkflowParallelStepNotificationsRequest, 1)
	go func() {
		cmd, err := stream.Recv()
		respChan <- testWorkflowParallelStepNotificationsRequest{resp: cmd, err: err}
	}()

	var cmd *cloud.TestWorkflowParallelStepNotificationsRequest
	select {
	case resp := <-respChan:
		cmd = resp.resp
		err := resp.err

		if err != nil {
			ag.logger.Errorf("agent stream receive: %v", err)
			return nil, err
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return cmd, nil
}

type testWorkflowParallelStepNotificationsRequest struct {
	resp *cloud.TestWorkflowParallelStepNotificationsRequest
	err  error
}

func (ag *Agent) sendTestWorkflowNotificationsResponse(ctx context.Context, stream cloud.TestKubeCloudAPI_GetTestWorkflowNotificationsStreamClient, resp *cloud.TestWorkflowNotificationsResponse) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- stream.Send(resp)
		close(errChan)
	}()

	t := time.NewTimer(ag.sendTimeout)
	select {
	case err := <-errChan:
		if !t.Stop() {
			<-t.C
		}
		return err
	case <-ctx.Done():
		if !t.Stop() {
			<-t.C
		}

		return ctx.Err()
	case <-t.C:
		return errors.New("send response too slow")
	}
}

func (ag *Agent) sendTestWorkflowServiceNotificationsResponse(ctx context.Context, stream cloud.TestKubeCloudAPI_GetTestWorkflowServiceNotificationsStreamClient, resp *cloud.TestWorkflowServiceNotificationsResponse) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- stream.Send(resp)
		close(errChan)
	}()

	t := time.NewTimer(ag.sendTimeout)
	select {
	case err := <-errChan:
		if !t.Stop() {
			<-t.C
		}
		return err
	case <-ctx.Done():
		if !t.Stop() {
			<-t.C
		}

		return ctx.Err()
	case <-t.C:
		return errors.New("send response too slow")
	}
}

func (ag *Agent) sendTestWorkflowParallelStepNotificationsResponse(ctx context.Context, stream cloud.TestKubeCloudAPI_GetTestWorkflowParallelStepNotificationsStreamClient, resp *cloud.TestWorkflowParallelStepNotificationsResponse) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- stream.Send(resp)
		close(errChan)
	}()

	t := time.NewTimer(ag.sendTimeout)
	select {
	case err := <-errChan:
		if !t.Stop() {
			<-t.C
		}
		return err
	case <-ctx.Done():
		if !t.Stop() {
			<-t.C
		}

		return ctx.Err()
	case <-t.C:
		return errors.New("send response too slow")
	}
}

func GetTestWorkflowNotificationsStream(testWorkflowResultsRepository testworkflow.Repository, executionWorker executionworkertypes.Worker) func(
	ctx context.Context, executionID string) (<-chan testkube.TestWorkflowExecutionNotification, error) {
	return func(ctx context.Context, executionID string) (<-chan testkube.TestWorkflowExecutionNotification, error) {
		execution, err := testWorkflowResultsRepository.Get(ctx, executionID)
		if err != nil {
			return nil, err
		}
		notifications := executionWorker.Notifications(ctx, execution.Id, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{
				Namespace:   execution.Namespace,
				Signature:   execution.Signature,
				ScheduledAt: common.Ptr(execution.ScheduledAt),
			},
		})
		if notifications.Err() != nil {
			return nil, notifications.Err()
		}
		return notifications.Channel(), nil
	}
}

func GetTestWorkflowServiceNotificationsStream(testWorkflowResultsRepository testworkflow.Repository, executionWorker executionworkertypes.Worker) func(
	ctx context.Context, executionID, serviceName string, serviceIndex int) (<-chan testkube.TestWorkflowExecutionNotification, error) {
	return func(ctx context.Context, executionID, serviceName string, serviceIndex int) (<-chan testkube.TestWorkflowExecutionNotification, error) {
		execution, err := testWorkflowResultsRepository.Get(ctx, executionID)
		if err != nil {
			return nil, err
		}

		if execution.Result != nil && execution.Result.IsFinished() {
			return nil, errors.New("test workflow execution is finished")
		}

		notifications := executionWorker.Notifications(ctx, fmt.Sprintf("%s-%s-%d", execution.Id, serviceName, serviceIndex), executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{
				Namespace:   execution.Namespace,
				ScheduledAt: common.Ptr(execution.ScheduledAt),
			},
		})
		if notifications.Err() != nil {
			return nil, notifications.Err()
		}
		return notifications.Channel(), nil
	}
}

func GetTestWorkflowParallelStepNotificationsStream(testWorkflowResultsRepository testworkflow.Repository, executionWorker executionworkertypes.Worker) func(
	ctx context.Context, executionID, ref string, workerIndex int) (<-chan testkube.TestWorkflowExecutionNotification, error) {
	return func(ctx context.Context, executionID, ref string, workerIndex int) (<-chan testkube.TestWorkflowExecutionNotification, error) {
		execution, err := testWorkflowResultsRepository.Get(ctx, executionID)
		if err != nil {
			return nil, err
		}

		if execution.Result != nil && execution.Result.IsFinished() {
			return nil, errors.New("test workflow execution is finished")
		}

		notifications := executionWorker.Notifications(ctx, fmt.Sprintf("%s-%s-%d", execution.Id, ref, workerIndex), executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{
				Namespace:   execution.Namespace,
				ScheduledAt: common.Ptr(execution.ScheduledAt),
			},
		})
		if notifications.Err() != nil {
			return nil, notifications.Err()
		}
		return notifications.Channel(), nil
	}
}

func GetDeprecatedLogStream(ctx context.Context, executionID string) (chan output.Output, error) {
	return nil, errors.New("deprecated features have been disabled")
}
