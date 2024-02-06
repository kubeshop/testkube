package adapter

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/pb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestCloudAdapter(t *testing.T) {

	t.Run("GRPC server receives log data", func(t *testing.T) {
		ts := NewTestServer().WithRandomPort()
		go ts.Run()

		ctx := context.Background()
		id := "id1"

		grpcConn, err := agent.NewGRPCConnection(ctx, true, true, ts.Url, log.DefaultLogger)
		assert.NoError(t, err)
		defer grpcConn.Close()
		grpcClient := pb.NewCloudLogsServiceClient(grpcConn)
		a := NewCloudAdapter(grpcClient, "APIKEY")

		err = a.Init(ctx, id)
		assert.NoError(t, err)
		err = a.Notify(ctx, id, *events.NewLog("log1"))
		assert.NoError(t, err)
		err = a.Notify(ctx, id, *events.NewLog("log2"))
		assert.NoError(t, err)
		err = a.Notify(ctx, id, *events.NewLog("log3"))
		assert.NoError(t, err)
		err = a.Notify(ctx, id, *events.NewLog("log4"))
		assert.NoError(t, err)
		err = a.Stop(ctx, id)
		assert.NoError(t, err)

		// cooldown
		time.Sleep(time.Millisecond * 100)

		assert.Len(t, ts.Received[id], 4)
	})

	t.Run("cleaning GRPC connections in adapter on Stop", func(t *testing.T) {
		ts := NewTestServer().WithRandomPort()
		go ts.Run()

		ctx := context.Background()
		id1 := "id1"
		id2 := "id2"
		id3 := "id3"

		grpcConn, err := agent.NewGRPCConnection(ctx, true, true, ts.Url, log.DefaultLogger)
		assert.NoError(t, err)
		defer grpcConn.Close()
		grpcClient := pb.NewCloudLogsServiceClient(grpcConn)
		a := NewCloudAdapter(grpcClient, "APIKEY")

		// send 3 streams
		err = a.Init(ctx, id1)
		assert.NoError(t, err)
		err = a.Notify(ctx, id1, *events.NewLog("log1"))
		assert.NoError(t, err)
		err = a.Stop(ctx, id1)
		assert.NoError(t, err)

		err = a.Init(ctx, id2)
		assert.NoError(t, err)
		err = a.Notify(ctx, id2, *events.NewLog("log2"))
		assert.NoError(t, err)
		err = a.Stop(ctx, id2)
		assert.NoError(t, err)

		err = a.Init(ctx, id3)
		assert.NoError(t, err)
		err = a.Notify(ctx, id3, *events.NewLog("log3"))
		assert.NoError(t, err)
		err = a.Stop(ctx, id3)
		assert.NoError(t, err)

		// cooldown
		time.Sleep(time.Millisecond * 100)

		assert.Len(t, ts.Received[id1], 1)
		assert.Len(t, ts.Received[id2], 1)
		assert.Len(t, ts.Received[id3], 1)

		// no stream are registered anymore
		assertNoStreams(t, a)
	})

	t.Run("Send and receive a lot of messages", func(t *testing.T) {
		ts := NewTestServer().WithRandomPort()
		go ts.Run()

		ctx := context.Background()
		id := "id1M"

		grpcConn, err := agent.NewGRPCConnection(ctx, true, true, ts.Url, log.DefaultLogger)
		assert.NoError(t, err)
		defer grpcConn.Close()
		grpcClient := pb.NewCloudLogsServiceClient(grpcConn)
		a := NewCloudAdapter(grpcClient, "APIKEY")

		err = a.Init(ctx, id)
		assert.NoError(t, err)

		messageCount := 10_000
		for i := 0; i < messageCount; i++ {
			err = a.Notify(ctx, id, *events.NewLog("log1"))
			assert.NoError(t, err)
		}

		// cooldown
		time.Sleep(time.Millisecond * 100)

		assert.Len(t, ts.Received[id], messageCount)
	})

}

func assertNoStreams(t *testing.T, a *CloudAdapter) {
	// no stream are registered anymore
	count := 0
	a.streams.Range(func(key, value any) bool {
		count++
		return true
	})
	assert.Equal(t, count, 0)
}

// Cloud Logs server mock
func NewTestServer() *TestServer {
	return &TestServer{
		Received: make(map[string][]*pb.Log),
	}
}

type TestServer struct {
	Url string
	pb.UnimplementedCloudLogsServiceServer
	Received map[string][]*pb.Log
}

func getVal(ctx context.Context, key string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "api-key header is missing")
	}
	apiKeyMeta := md.Get(key)
	if len(apiKeyMeta) != 1 {
		return "", status.Error(codes.Unauthenticated, "api-key header is empty")
	}
	if apiKeyMeta[0] == "" {
		return "", status.Error(codes.Unauthenticated, "api-key header value is empty")
	}

	return apiKeyMeta[0], nil
}

func (s *TestServer) Stream(stream pb.CloudLogsService_StreamServer) error {
	ctx := stream.Context()
	v, err := getVal(ctx, "execution-id")
	if err != nil {
		return err
	}
	id := v
	s.Received[id] = []*pb.Log{}

	for {
		in, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				err := stream.SendAndClose(&pb.StreamResponse{Message: "completed"})
				if err != nil {
					return status.Error(codes.Internal, "can't close stream: "+err.Error())
				}
				return nil
			}
			return status.Error(codes.Internal, "can't receive stream: "+err.Error())
		}

		s.Received[id] = append(s.Received[id], in)
	}
}

func (s *TestServer) WithRandomPort() *TestServer {
	port := rand.Intn(1000) + 18000
	s.Url = fmt.Sprintf("127.0.0.1:%d", port)
	return s
}

func (s *TestServer) Run() (err error) {
	lis, err := net.Listen("tcp", s.Url)
	if err != nil {
		return errors.Errorf("net listen: %v", err)
	}

	var opts []grpc.ServerOption
	creds := insecure.NewCredentials()
	opts = append(opts, grpc.Creds(creds), grpc.MaxRecvMsgSize(math.MaxInt32))

	// register server logs
	srv := grpc.NewServer(opts...)
	srv.RegisterService(&pb.CloudLogsService_ServiceDesc, s)
	srv.Serve(lis)

	if err != nil {
		return errors.Wrap(err, "grpc server error")
	}
	return nil
}
