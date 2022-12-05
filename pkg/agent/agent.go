package agent

import (
	"context"
	"fmt"
	"io"

	"github.com/kubeshop/testkube/internal/pkg/api"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	apiKey             = "api-key"
	healthcheckCommand = "healthcheck"
)

type Agent struct {
	conn    *grpc.ClientConn
	client  cloud.TestKubeCloudAPIClient
	handler fasthttp.RequestHandler
	logger  *zap.SugaredLogger
	apiKey  string

	events chan testkube.Event
}

func NewAgent(logger *zap.SugaredLogger, handler fasthttp.RequestHandler, server, apiKey string, isInsecure bool) (*Agent, error) {
	var security grpc.DialOption
	if isInsecure {
		security = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		security = grpc.WithTransportCredentials(credentials.NewTLS(nil))
	}

	conn, err := grpc.Dial(server, grpc.WithBlock(), grpc.WithUserAgent(api.Version+"/"+api.Commit), security)
	if err != nil {
		return nil, err
	}

	client := cloud.NewTestKubeCloudAPIClient(conn)
	return &Agent{conn: conn,
		client:  client,
		handler: handler,
		logger:  logger,
		apiKey:  apiKey,
		events:  make(chan testkube.Event),
	}, nil
}

func (ag *Agent) Run(ctx context.Context) error {
	var opts []grpc.CallOption
	md := metadata.Pairs(apiKey, ag.apiKey)
	ctx = metadata.NewOutgoingContext(ctx, md)

	//TODO figure out how to retry this method in case of network failure

	// creates a new Stream from the client side. ctx is used for the lifetime of the stream.
	stream, err := ag.client.Execute(ctx, opts...)
	if err != nil {
		ag.logger.Errorf("failed to execute: %w", err)
		return fmt.Errorf("failed to setup stream: %w", err)
	}

	for {
		cmd, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			ag.logger.Errorf("agent stream recv: %w", err)
			return err
		}

		switch {
		case cmd.Url == healthcheckCommand:
			resp := &cloud.ExecuteResponse{Status: 0}
			err = stream.Send(resp)
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
			err = stream.Send(resp)
			if err != nil {
				ag.logger.Errorf("error stream send: %w", err)
				return err
			}
		}
	}

	return nil
}

func (ag *Agent) Close() error {
	return ag.conn.Close()
}
