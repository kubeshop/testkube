package controlplaneclient

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

var (
	grpcOpts = []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
)

// TODO: add timeout?
func call[Request any, Response any](ctx context.Context, md metadata.MD, fn func(context.Context, Request, ...grpc.CallOption) (Response, error), req Request) (Response, error) {
	if ctx.Err() != nil {
		var v Response
		return v, ctx.Err()
	}
	return fn(metadata.NewOutgoingContext(ctx, md), req, grpcOpts...)
}

// TODO: add timeout?
func watch[Response any](ctx context.Context, md metadata.MD, fn func(context.Context, ...grpc.CallOption) (Response, error)) (Response, error) {
	if ctx.Err() != nil {
		var v Response
		return v, ctx.Err()
	}
	return fn(metadata.NewOutgoingContext(ctx, md), grpcOpts...)
}

func getGrpcErrorCode(err error) codes.Code {
	if err == nil {
		return codes.Unknown
	}
	if e, ok := err.(interface{ GRPCStatus() *status.Status }); ok {
		return e.GRPCStatus().Code()
	}
	return codes.Unknown
}

type notificationRequest interface {
	GetRequestType() cloud.TestWorkflowNotificationsRequestType
	GetStreamId() string
}

type notificationSrv[Request any, Response any] interface {
	Send(Response) error
	Recv() (Request, error)
}

func processNotifications[Request notificationRequest, Response any, Srv notificationSrv[Request, Response]](
	ctx context.Context,
	md metadata.MD,
	fn func(context.Context, ...grpc.CallOption) (Srv, error),
	buildPongNotification func(streamId string) Response,
	buildNotification func(streamId string, seqNo uint32, notification *testkube.TestWorkflowExecutionNotification) Response,
	buildError func(streamId string, message string) Response,
	process func(ctx context.Context, req Request) NotificationWatcher,
	sendTimeout time.Duration,
) error {
	g, ctx := errgroup.WithContext(ctx)
	stream, err := watch(ctx, md, fn)
	if err != nil {
		return err
	}

	responses := make(chan Response, 5)

	// Send responses in sequence
	// GRPC stream have special requirements for concurrency on SendMsg, and RecvMsg calls.
	// Please check https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	g.Go(func() error {
		for msg := range responses {
			errChan := make(chan error, 1)
			go func() {
				errChan <- stream.Send(msg)
				close(errChan)
			}()

			t := time.NewTimer(sendTimeout)

			select {
			case err := <-errChan:
				if !t.Stop() {
					<-t.C
				}
				if err != nil {
					return err
				}
			case <-ctx.Done():
				if !t.Stop() {
					<-t.C
				}
				return ctx.Err()
			case <-t.C:
				return fmt.Errorf("send response too slow")
			}
		}
		return nil
	})

	// Process the requests
	g.Go(func() error {
		var wg sync.WaitGroup
		defer func() {
			wg.Wait()
			close(responses)
		}()
		for {
			// Take the context error if possible
			if err == nil && ctx.Err() != nil {
				err = ctx.Err()
			}

			// Handle the error
			if err != nil {
				return err
			}

			// Get the next request
			var req Request
			req, err = stream.Recv()
			if err != nil {
				continue
			}

			// Send PONG to the PING message
			if req.GetRequestType() == cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK {
				responses <- buildPongNotification(req.GetStreamId())
				continue
			}

			// Start reading the notifications
			wg.Add(1)
			g.Go(func(req Request) func() error {
				return func() error {
					defer wg.Done()
					seqNo := uint32(0)
					watcher := process(ctx, req) // TODO: Make ctx per request, so the stream could be stopped
					for notification := range watcher.Channel() {
						responses <- buildNotification(req.GetStreamId(), seqNo, notification)
						seqNo++
					}
					if watcher.Err() != nil {
						responses <- buildError(req.GetStreamId(), watcher.Err().Error())
					}
					return nil
				}
			}(req))
		}
	})

	return g.Wait()
}

func buildCloudNotification(streamId string, seqNo uint32, notification *testkube.TestWorkflowExecutionNotification) *cloud.TestWorkflowNotificationsResponse {
	response := &cloud.TestWorkflowNotificationsResponse{
		StreamId:  streamId,
		SeqNo:     seqNo,
		Timestamp: notification.Ts.Format(time.RFC3339Nano),
		Ref:       notification.Ref,
	}
	if notification.Result != nil {
		m, _ := json.Marshal(notification.Result)
		response.Type = cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_RESULT
		response.Message = string(m)
	} else if notification.Output != nil {
		m, _ := json.Marshal(notification.Output)
		response.Type = cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_OUTPUT
		response.Message = string(m)
	} else {
		response.Type = cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_LOG
		response.Message = notification.Log
	}
	return response
}

func buildCloudError(streamId string, message string) *cloud.TestWorkflowNotificationsResponse {
	ts := time.Now()
	return &cloud.TestWorkflowNotificationsResponse{
		StreamId:  streamId,
		SeqNo:     0,
		Timestamp: ts.Format(time.RFC3339Nano),
		Type:      cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_ERROR,
		Message:   fmt.Sprintf("%s %s", ts.Format(constants.PreciseTimeFormat), message),
	}
}

func convertCloudResponseToService(response *cloud.TestWorkflowNotificationsResponse) *cloud.TestWorkflowServiceNotificationsResponse {
	return &cloud.TestWorkflowServiceNotificationsResponse{
		StreamId:  response.StreamId,
		SeqNo:     response.SeqNo,
		Timestamp: response.Timestamp,
		Ref:       response.Ref,
		Type:      response.Type,
		Message:   response.Message,
	}
}

func buildServiceCloudNotification(streamId string, seqNo uint32, notification *testkube.TestWorkflowExecutionNotification) *cloud.TestWorkflowServiceNotificationsResponse {
	return convertCloudResponseToService(buildCloudNotification(streamId, seqNo, notification))
}

func buildServiceCloudError(streamId string, message string) *cloud.TestWorkflowServiceNotificationsResponse {
	return convertCloudResponseToService(buildCloudError(streamId, message))
}

func buildPongNotification(streamId string) *cloud.TestWorkflowNotificationsResponse {
	return &cloud.TestWorkflowNotificationsResponse{StreamId: streamId, SeqNo: 0}
}

func buildParallelStepPongNotification(streamId string) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return &cloud.TestWorkflowParallelStepNotificationsResponse{StreamId: streamId, SeqNo: 0}
}

func buildServicePongNotification(streamId string) *cloud.TestWorkflowServiceNotificationsResponse {
	return &cloud.TestWorkflowServiceNotificationsResponse{StreamId: streamId, SeqNo: 0}
}

func convertCloudResponseToParallelStep(response *cloud.TestWorkflowNotificationsResponse) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return &cloud.TestWorkflowParallelStepNotificationsResponse{
		StreamId:  response.StreamId,
		SeqNo:     response.SeqNo,
		Timestamp: response.Timestamp,
		Ref:       response.Ref,
		Type:      response.Type,
		Message:   response.Message,
	}
}

func buildParallelStepCloudNotification(streamId string, seqNo uint32, notification *testkube.TestWorkflowExecutionNotification) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return convertCloudResponseToParallelStep(buildCloudNotification(streamId, seqNo, notification))
}

func buildParallelStepCloudError(streamId string, message string) *cloud.TestWorkflowParallelStepNotificationsResponse {
	return convertCloudResponseToParallelStep(buildCloudError(streamId, message))
}
