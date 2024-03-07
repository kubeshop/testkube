package agent_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/featureflags"
)

func TestEventLoop(t *testing.T) {
	url := "localhost:8998"

	ctx, cancel := context.WithCancel(context.Background())
	cloudSrv := newEventServer(ctx)

	go func() {
		lis, err := net.Listen("tcp", url)
		if err != nil {
			panic(err)
		}

		var opts []grpc.ServerOption
		grpcServer := grpc.NewServer(opts...)
		cloud.RegisterTestKubeCloudAPIServer(grpcServer, cloudSrv)
		err = grpcServer.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	logger, _ := zap.NewDevelopment()

	grpcConn, err := agent.NewGRPCConnection(context.Background(), true, false, url, "", "", "", log.DefaultLogger)
	ui.ExitOnError("error creating gRPC connection", err)
	defer grpcConn.Close()

	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)

	var logStreamFunc func(ctx context.Context, executionID string) (chan output.Output, error)
	var workflowNotificationsStreamFunc func(ctx context.Context, executionID string) (chan testkube.TestWorkflowExecutionNotification, error)

	proContext := config.ProContext{APIKey: "api-key", WorkerCount: 5, LogStreamWorkerCount: 5, WorkflowNotificationsWorkerCount: 5}
	agent, err := agent.NewAgent(logger.Sugar(), nil, grpcClient, logStreamFunc, workflowNotificationsStreamFunc, "", "", nil, featureflags.FeatureFlags{}, proContext)
	assert.NoError(t, err)
	go func() {
		l, err := agent.Load()
		if err != nil {
			panic(err)
		}

		var i int
		for {
			res := l[0].Notify(testkube.Event{Id: fmt.Sprintf("%d", i)})
			if res.Error_ != "" {
				continue
			}
		}
	}()

	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return agent.Run(groupCtx)
	})

	time.Sleep(100 * time.Millisecond)
	cancel()

	g.Wait()

	assert.True(t, cloudSrv.Count() >= 5)
}

func (cws *CloudEventServer) Count() int {
	return cws.messageCount
}

func (cws *CloudEventServer) ExecuteAsync(srv cloud.TestKubeCloudAPI_ExecuteAsyncServer) error {
	<-cws.ctx.Done()

	return nil
}

func (cws *CloudEventServer) GetLogsStream(srv cloud.TestKubeCloudAPI_GetLogsStreamServer) error {
	<-cws.ctx.Done()

	return nil
}

func (cws *CloudEventServer) GetTestWorkflowNotificationsStream(srv cloud.TestKubeCloudAPI_GetTestWorkflowNotificationsStreamServer) error {
	<-cws.ctx.Done()

	return nil
}

func (cws *CloudEventServer) Send(srv cloud.TestKubeCloudAPI_SendServer) error {
	md, ok := metadata.FromIncomingContext(srv.Context())
	if !ok {
		panic("no metadata")
	}
	apiKey := md.Get("api-key")
	if apiKey[0] != "api-key" {
		panic("error bad api-key")
	}

	for {
		if srv.Context().Err() != nil {
			return srv.Context().Err()
		}
		resp, err := srv.Recv()
		if err != nil {
			return err
		}

		if resp.Opcode == cloud.Opcode_HEALTH_CHECK {
			continue
		}

		if resp.Opcode != cloud.Opcode_TEXT_FRAME {
			panic("bad opcode")
		}
		cws.messageCount++

		if cws.messageCount >= 5 {
			return nil
		}
	}
}

func newEventServer(ctx context.Context) *CloudEventServer {
	return &CloudEventServer{ctx: ctx}
}

type CloudEventServer struct {
	cloud.UnimplementedTestKubeCloudAPIServer
	messageCount int
	ctx          context.Context
}
