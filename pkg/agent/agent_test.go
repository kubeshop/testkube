package agent_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestRun(t *testing.T) {
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

	m := func(ctx *fasthttp.RequestCtx) {
		h := &ctx.Response.Header
		h.Add("Content-type", "application/json")
		fmt.Fprintf(ctx, "Hi there! RequestURI is %q", ctx.RequestURI())
	}

	logger, _ := zap.NewDevelopment()
	agent, err := agent.NewAgent(logger.Sugar(), m, url, "api-key", true)
	if err != nil {
		t.Fatal(err)
	}

	err = agent.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
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
func newServer() *CloudServer {

	return &CloudServer{}
}
