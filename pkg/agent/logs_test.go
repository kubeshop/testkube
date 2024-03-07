package agent_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestLogStream(t *testing.T) {
	url := "localhost:8997"

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		lis, err := net.Listen("tcp", url)
		if err != nil {
			panic(err)
		}

		var opts []grpc.ServerOption
		grpcServer := grpc.NewServer(opts...)
		cloud.RegisterTestKubeCloudAPIServer(grpcServer, newLogStreamServer(ctx))
		grpcServer.Serve(lis)
	}()

	m := func(ctx *fasthttp.RequestCtx) {
		h := &ctx.Response.Header
		h.Add("Content-type", "application/json")
		fmt.Fprintf(ctx, "Hi there! RequestURI is %q", ctx.RequestURI())
	}

	grpcConn, err := agent.NewGRPCConnection(context.Background(), true, false, url, "", "", "", log.DefaultLogger)
	ui.ExitOnError("error creating gRPC connection", err)
	defer grpcConn.Close()

	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)

	var msgCnt int32
	logStreamFunc := func(ctx context.Context, executionID string) (chan output.Output, error) {
		ch := make(chan output.Output, 5)

		ch <- output.Output{
			Type_:   output.TypeLogLine,
			Content: "log1",
			Time:    time.Now(),
		}
		msgCnt++
		return ch, nil
	}
	var workflowNotificationsStreamFunc func(ctx context.Context, executionID string) (chan testkube.TestWorkflowExecutionNotification, error)

	logger, _ := zap.NewDevelopment()
	proContext := config.ProContext{APIKey: "api-key", WorkerCount: 5, LogStreamWorkerCount: 5, WorkflowNotificationsWorkerCount: 5}
	agent, err := agent.NewAgent(logger.Sugar(), m, grpcClient, logStreamFunc, workflowNotificationsStreamFunc, "", "", nil, featureflags.FeatureFlags{}, proContext)
	if err != nil {
		t.Fatal(err)
	}

	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return agent.Run(groupCtx)
	})

	time.Sleep(100 * time.Millisecond)
	cancel()

	g.Wait()

	assert.True(t, msgCnt > 0)
}

type CloudLogsServer struct {
	ctx context.Context
	cloud.UnimplementedTestKubeCloudAPIServer
}

func (cs *CloudLogsServer) ExecuteAsync(srv cloud.TestKubeCloudAPI_ExecuteAsyncServer) error {
	<-cs.ctx.Done()
	return nil
}

func (cs *CloudLogsServer) GetTestWorkflowNotificationsStream(srv cloud.TestKubeCloudAPI_GetTestWorkflowNotificationsStreamServer) error {
	<-cs.ctx.Done()
	return nil
}

func (cs *CloudLogsServer) GetLogsStream(srv cloud.TestKubeCloudAPI_GetLogsStreamServer) error {
	md, ok := metadata.FromIncomingContext(srv.Context())
	if !ok {
		panic("no metadata")
	}
	apiKey := md.Get("api-key")
	if apiKey[0] != "api-key" {
		panic("error bad api-key")
	}

	req := &cloud.LogsStreamRequest{
		StreamId:    "streamID",
		ExecutionId: "executionID",
	}
	err := srv.Send(req)
	if err != nil {
		return err
	}

	resp, err := srv.Recv()
	if err != nil {
		return err
	}
	fmt.Println(resp)

	return nil
}

func newLogStreamServer(ctx context.Context) *CloudLogsServer {
	return &CloudLogsServer{ctx: ctx}
}
