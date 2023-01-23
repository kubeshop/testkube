package agent_test

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/cloud"
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
		cloud.RegisterTestKubeCloudAPIServer(grpcServer, newServer())
		grpcServer.Serve(lis)
	}()

	var msgCnt int32
	m := func(ctx *fasthttp.RequestCtx) {
		h := &ctx.Response.Header
		h.Add("Content-type", "application/json")
		fmt.Fprintf(ctx, "Hi there! RequestURI is %q", ctx.RequestURI())
		atomic.AddInt32(&msgCnt, 1)
	}

	grpcConn, err := agent.NewGRPCConnection(context.Background(), true, url, log.DefaultLogger)
	ui.ExitOnError("error creating gRPC connection", err)
	defer grpcConn.Close()

	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)

	logger, _ := zap.NewDevelopment()
	agent, err := agent.NewAgent(logger.Sugar(), m, "api-key", grpcClient)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	g, groupCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return agent.Run(groupCtx)
	})

	time.Sleep(100 * time.Millisecond)
	cancel()

	g.Wait()

	assert.True(t, msgCnt > 0)
}

type CloudServer struct {
	cloud.UnimplementedTestKubeCloudAPIServer
}

func (cs *CloudServer) Execute(srv cloud.TestKubeCloudAPI_ExecuteServer) error {
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
func newServer() *CloudServer {
	return &CloudServer{}
}
