package agent

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/kubeshop/testkube/pkg/cloud"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

const logStreamRetryCount = 10

func (ag *Agent) runLogStreamLoop(ctx context.Context) error {
	ctx = AddAPIKeyMeta(ctx, ag.apiKey)

	ag.logger.Infow("initiating log streaming connection with Cloud API")
	// creates a new Stream from the client side. ctx is used for the lifetime of the stream.
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	stream, err := ag.client.GetLogsStream(ctx, opts...)
	if err != nil {
		ag.logger.Errorf("failed to execute: %w", err)
		return errors.Wrap(err, "failed to setup stream")
	}

	// GRPC stream have special requirements for concurrency on SendMsg, and RecvMsg calls.
	// Please check https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			cmd, err := ag.receiveLogStreamRequest(groupCtx, stream)
			if err != nil {
				return err
			}

			ag.logStreamRequestBuffer <- cmd
		}
	})

	g.Go(func() error {
		for {
			select {
			case resp := <-ag.logStreamResponseBuffer:
				err := ag.sendLogStreamResponse(groupCtx, stream, resp)
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

func (ag *Agent) runLogStreamWorker(ctx context.Context, numWorkers int) error {
	g, groupCtx := errgroup.WithContext(ctx)
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for {
				select {
				case req := <-ag.logStreamRequestBuffer:

					if req.RequestType == cloud.LogsStreamRequestType_STREAM_HEALTH_CHECK {
						ag.logStreamResponseBuffer <- &cloud.LogsStreamResponse{
							StreamId: req.StreamId,
							SeqNo:    0,
						}
						break
					}

					err := ag.executeLogStreamRequest(groupCtx, req)
					if err != nil {
						ag.logger.Errorf("error executing log stream request: %s", err.Error())
					}
				case <-groupCtx.Done():
					return groupCtx.Err()
				}
			}
		})
	}
	return g.Wait()
}

func (ag *Agent) executeLogStreamRequest(ctx context.Context, req *cloud.LogsStreamRequest) error {
	logCh, err := ag.logStreamFunc(ctx, req.ExecutionId)
	for i := 0; i < logStreamRetryCount; i++ {
		if err != nil {
			// We have a race condition here
			// Cloud sometimes slow to insert execution or test
			// while LogStream request from websockets comes in faster
			// so we retry up to logStreamRetryCount times.
			time.Sleep(100 * time.Millisecond)
			logCh, err = ag.logStreamFunc(ctx, req.ExecutionId)
		}
	}
	if err != nil {
		ag.logStreamResponseBuffer <- &cloud.LogsStreamResponse{
			StreamId:   req.StreamId,
			SeqNo:      0,
			LogMessage: fmt.Sprintf("cannot get pod logs: %s", err.Error()),
			IsError:    true,
		}
		return nil
	}

	for {
		var i int64
		select {
		case logOutput, ok := <-logCh:
			if !ok {
				return nil
			}
			msg := &cloud.LogsStreamResponse{
				StreamId:   req.StreamId,
				SeqNo:      i,
				LogMessage: logOutput.String(),
			}
			i++

			select {
			case ag.logStreamResponseBuffer <- msg:
			case <-ctx.Done():
				return ctx.Err()
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (ag *Agent) receiveLogStreamRequest(ctx context.Context, stream cloud.TestKubeCloudAPI_GetLogsStreamClient) (*cloud.LogsStreamRequest, error) {
	respChan := make(chan logsStreamRequest, 1)
	go func() {
		cmd, err := stream.Recv()
		respChan <- logsStreamRequest{resp: cmd, err: err}
	}()

	var cmd *cloud.LogsStreamRequest
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

type logsStreamRequest struct {
	resp *cloud.LogsStreamRequest
	err  error
}

func (ag *Agent) sendLogStreamResponse(ctx context.Context, stream cloud.TestKubeCloudAPI_GetLogsStreamClient, resp *cloud.LogsStreamResponse) error {
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
