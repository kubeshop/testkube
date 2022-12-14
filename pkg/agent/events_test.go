package agent_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestRunEventLoop(t *testing.T) {
	url := "localhost:8998"
	cloudSrv := newEventServer()

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

	agent, err := agent.NewAgent(logger.Sugar(), nil, url, "api-key", true)
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

	agent.RunEventLoop(context.Background())

	assert.Equal(t, cloudSrv.Count(), 5)
}

func (cws *CloudEventServer) Count() int {
	return cws.messageCount
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
		resp, err := srv.Recv()
		if err != nil {
			return err
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

func newEventServer() *CloudEventServer {
	return &CloudEventServer{}
}

type CloudEventServer struct {
	cloud.UnimplementedTestKubeCloudAPIServer
	messageCount int
}
