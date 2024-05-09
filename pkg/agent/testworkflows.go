package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowcontroller"
)

const testWorkflowNotificationsRetryCount = 10

func getTestWorkflowNotificationType(n testkube.TestWorkflowExecutionNotification) cloud.TestWorkflowNotificationType {
	if n.Result != nil {
		return cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_RESULT
	} else if n.Output != nil {
		return cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_OUTPUT
	}
	return cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_LOG
}

func (ag *Agent) runTestWorkflowNotificationsLoop(ctx context.Context) error {
	ctx = AddAPIKeyMeta(ctx, ag.apiKey)

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

func (ag *Agent) executeWorkflowNotificationsRequest(ctx context.Context, req *cloud.TestWorkflowNotificationsRequest) error {
	notificationsCh, err := ag.testWorkflowNotificationsFunc(ctx, req.ExecutionId)
	for i := 0; i < testWorkflowNotificationsRetryCount; i++ {
		if err != nil {
			// We have a race condition here
			// Cloud sometimes slow to insert execution or test
			// while WorkflowNotifications request from websockets comes in faster
			// so we retry up to testWorkflowNotificationsRetryCount times.
			time.Sleep(100 * time.Millisecond)
			notificationsCh, err = ag.testWorkflowNotificationsFunc(ctx, req.ExecutionId)
		}
	}
	if err != nil {
		message := fmt.Sprintf("cannot get pod logs: %s", err.Error())
		ag.testWorkflowNotificationsResponseBuffer <- &cloud.TestWorkflowNotificationsResponse{
			StreamId: req.StreamId,
			SeqNo:    0,
			Type:     cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_ERROR,
			Message:  fmt.Sprintf("%s %s", time.Now().Format(testworkflowcontroller.KubernetesLogTimeFormat), message),
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
