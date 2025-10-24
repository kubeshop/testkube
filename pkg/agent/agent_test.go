package agent_test

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/featureflags"
)

func TestCommandExecution(t *testing.T) {
	url := "localhost:9999"

	go func() {
		lis, err := net.Listen("tcp", url)
		if err != nil {
			panic(err)
		}

		var opts []grpc.ServerOption
		grpcServer := grpc.NewServer(opts...)
		cloud.RegisterTestKubeCloudAPIServer(grpcServer, newServer(t.Context()))
		grpcServer.Serve(lis)
	}()

	var msgCnt int32
	m := func(ctx *fasthttp.RequestCtx) {
		h := &ctx.Response.Header
		h.Add("Content-type", "application/json")
		fmt.Fprintf(ctx, "Hi there! RequestURI is %q", ctx.RequestURI())
		atomic.AddInt32(&msgCnt, 1)
	}

	grpcConn, err := agentclient.NewVeryInsecureGRPCClientDoNotUseThisClientUnlessYouAreReallySureYouKnowWhatYouAreDoing(context.Background(), true, url, log.DefaultLogger)
	ui.ExitOnError("error creating gRPC connection", err)
	defer grpcConn.Close()

	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)

	var logStreamFunc func(ctx context.Context, executionID string) (chan output.Output, error)

	logger, _ := zap.NewDevelopment()
	proContext := config.ProContext{APIKey: "api-key", WorkerCount: 5, LogStreamWorkerCount: 5}
	agent, err := agent.NewAgent(logger.Sugar(), m, grpcClient, logStreamFunc, "", "", featureflags.FeatureFlags{}, &proContext, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return agent.Run(groupCtx)
	})

	g.Wait()

	assert.True(t, msgCnt > 0)
}

type CloudServer struct {
	cloud.UnimplementedTestKubeCloudAPIServer
	ctx context.Context
}

func (cs *CloudServer) GetLogsStream(srv cloud.TestKubeCloudAPI_GetLogsStreamServer) error {
	<-cs.ctx.Done()

	return nil
}

func (cs *CloudServer) GetTestWorkflowNotificationsStream(srv cloud.TestKubeCloudAPI_GetTestWorkflowNotificationsStreamServer) error {
	<-cs.ctx.Done()

	return nil
}

func (cs *CloudServer) GetTestWorkflowServiceNotificationsStream(srv cloud.TestKubeCloudAPI_GetTestWorkflowServiceNotificationsStreamServer) error {
	<-cs.ctx.Done()

	return nil
}

func (cs *CloudServer) GetTestWorkflowParallelStepNotificationsStream(srv cloud.TestKubeCloudAPI_GetTestWorkflowParallelStepNotificationsStreamServer) error {
	<-cs.ctx.Done()

	return nil
}

func (cs *CloudServer) ExecuteAsync(srv cloud.TestKubeCloudAPI_ExecuteAsyncServer) error {
	md, ok := metadata.FromIncomingContext(srv.Context())
	if !ok {
		panic("no metadata")
	}
	apiKey := md.Get("api-key")
	if apiKey[0] != "api-key" {
		panic("error bad api-key")
	}

	req := &cloud.ExecuteRequest{Method: "url", Url: "localhost/v1/tests", Body: nil}
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

func (cs *CloudServer) Send(srv cloud.TestKubeCloudAPI_SendServer) error {
	for {
		if srv.Context().Err() != nil {
			return srv.Context().Err()
		}

		_, err := srv.Recv()
		if err != nil {
			return err
		}
	}

}
func newServer(ctx context.Context) *CloudServer {
	return &CloudServer{ctx: ctx}
}
