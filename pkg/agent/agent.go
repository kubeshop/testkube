package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/pkg/version"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

const (
	apiKeyMeta         = "api-key"
	healthcheckCommand = "healthcheck"
)

func NewGRPCConnection(ctx context.Context, isInsecure bool, server string, logger *zap.SugaredLogger) (*grpc.ClientConn, error) {
	creds := credentials.NewTLS(nil)
	if isInsecure {
		creds = insecure.NewCredentials()
	}

	userAgent := version.Version + "/" + version.Commit
	logger.Infow("initiating connection with Cloud API", "userAgent", userAgent)
	return grpc.DialContext(ctx, server, grpc.WithBlock(), grpc.WithUserAgent(userAgent), grpc.WithTransportCredentials(creds))
}

type Agent struct {
	client  cloud.TestKubeCloudAPIClient
	handler fasthttp.RequestHandler
	logger  *zap.SugaredLogger
	apiKey  string

	events              chan testkube.Event
	sendTimeout         time.Duration
	receiveTimeout      time.Duration
	healthcheckInterval time.Duration
}

func NewAgent(logger *zap.SugaredLogger, handler fasthttp.RequestHandler, apiKey string, client cloud.TestKubeCloudAPIClient) (*Agent, error) {
	return &Agent{
		handler:             handler,
		logger:              logger,
		apiKey:              apiKey,
		client:              client,
		events:              make(chan testkube.Event),
		receiveTimeout:      5 * time.Minute,
		sendTimeout:         30 * time.Second,
		healthcheckInterval: 30 * time.Second,
	}, nil
}

func (ag *Agent) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := ag.run(ctx)

		ag.logger.Errorw("agent connection failed, reconnecting", "error", err)

		// TODO: some smart back off strategy?
		time.Sleep(5 * time.Second)
	}
}

func (ag *Agent) run(ctx context.Context) (err error) {
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return ag.runCommandLoop(groupCtx)
	})

	g.Go(func() error {
		return ag.runEventLoop(groupCtx)
	})

	err = g.Wait()

	return err
}

func (ag *Agent) sendResponse(ctx context.Context, stream cloud.TestKubeCloudAPI_ExecuteClient, resp *cloud.ExecuteResponse) error {
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
		return errors.New("too slow")
	}
}

func (ag *Agent) receiveCommand(ctx context.Context, stream cloud.TestKubeCloudAPI_ExecuteClient) (*cloud.ExecuteRequest, error) {
	respChan := make(chan cloudResponse, 1)
	go func() {
		cmd, err := stream.Recv()
		respChan <- cloudResponse{resp: cmd, err: err}
	}()

	t := time.NewTimer(ag.receiveTimeout)
	var cmd *cloud.ExecuteRequest
	select {
	case resp := <-respChan:
		if !t.Stop() {
			<-t.C
		}

		cmd = resp.resp
		err := resp.err

		if err != nil {
			ag.logger.Errorf("agent stream recv: %v", err)
			return nil, err
		}
	case <-ctx.Done():
		if !t.Stop() {
			<-t.C
		}

		return nil, ctx.Err()
	case <-t.C:
		return nil, errors.New("too slow")
	}

	return cmd, nil
}

func (ag *Agent) runCommandLoop(ctx context.Context) error {
	ctx = AddAPIKeyMeta(ctx, ag.apiKey)

	//TODO figure out how to retry this method in case of network failure

	ag.logger.Infow("initiating streaming connection with Cloud API")
	// creates a new Stream from the client side. ctx is used for the lifetime of the stream.
	var opts []grpc.CallOption
	stream, err := ag.client.Execute(ctx, opts...)
	if err != nil {
		ag.logger.Errorf("failed to execute: %w", err)
		return errors.Wrap(err, "failed to setup stream")
	}

	for {
		cmd, err := ag.receiveCommand(ctx, stream)
		if err != nil {
			return err
		}
		switch {
		case cmd.Url == healthcheckCommand:
			resp := &cloud.ExecuteResponse{Status: 0}

			err = ag.sendResponse(ctx, stream, resp)
			if err != nil {
				ag.logger.Errorf("stream send: %w", err)
				return err
			}
		default:
			req := &fasthttp.RequestCtx{}
			r := fasthttp.AcquireRequest()
			r.Header.SetHost("localhost")
			r.Header.SetMethod(cmd.Method)

			for k, values := range cmd.Headers {
				for _, value := range values.Header {
					r.Header.Add(k, value)
				}
			}
			r.SetBody(cmd.Body)
			uri := &fasthttp.URI{}

			err = uri.Parse(nil, []byte(cmd.Url))
			if err != nil {
				ag.logger.Errorf("agent bad command url: %w", err)
				resp := &cloud.ExecuteResponse{Status: 400, Body: []byte(fmt.Sprintf("bad command url: %s", err))}
				if err := stream.Send(resp); err != nil {
					ag.logger.Errorf("stream send: %w", err)
				}
				return err
			}
			r.SetURI(uri)

			req.Init(r, nil, nil)
			ag.handler(req)

			fasthttp.ReleaseRequest(r)

			headers := make(map[string]*cloud.HeaderValue)
			req.Response.Header.VisitAll(func(key, value []byte) {
				_, ok := headers[string(key)]
				if !ok {
					headers[string(key)] = &cloud.HeaderValue{Header: []string{string(value)}}
					return
				}

				headers[string(key)].Header = append(headers[string(key)].Header, string(value))
			})

			resp := &cloud.ExecuteResponse{Headers: headers, Status: int64(req.Response.StatusCode()), Body: req.Response.Body()}

			err = ag.sendResponse(ctx, stream, resp)
			if err != nil {
				ag.logger.Errorf("error stream send: %w", err)
				return err
			}
		}
	}
}

func AddAPIKeyMeta(ctx context.Context, apiKey string) context.Context {
	md := metadata.Pairs(apiKeyMeta, apiKey)
	return metadata.NewOutgoingContext(ctx, md)
}

type cloudResponse struct {
	resp *cloud.ExecuteRequest
	err  error
}
