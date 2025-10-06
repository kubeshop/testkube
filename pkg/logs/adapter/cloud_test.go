package adapter

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/pb"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestCloudAdapter_Integration(t *testing.T) {
	test.IntegrationTest(t)

	t.Run("GRPC server receives log data", func(t *testing.T) {
		// given grpc test server
		ts := StartTestServer(t)

		ctx := context.Background()
		id := "id1"

		// and connection
		grpcConn, err := agentclient.NewGRPCConnection(ctx, true, true, ts.Url, "", log.DefaultLogger)
		if err != nil {
			t.Fatalf("Failed to create gRPC connection: %v", err)
		}
		t.Cleanup(func() {
			grpcConn.Close()
		})

		// and log stream client
		grpcClient := pb.NewCloudLogsServiceClient(grpcConn)
		a := NewCloudAdapter(grpcClient, "APIKEY")

		// when stream is initialized
		err = a.Init(ctx, id)
		assert.NoError(t, err)
		// and data is sent to it
		err = a.Notify(ctx, id, *events.NewLog("log1"))
		assert.NoError(t, err)
		err = a.Notify(ctx, id, *events.NewLog("log2"))
		assert.NoError(t, err)
		err = a.Notify(ctx, id, *events.NewLog("log3"))
		assert.NoError(t, err)
		err = a.Notify(ctx, id, *events.NewLog("log4"))
		assert.NoError(t, err)
		// and stream is stopped after sending logs to it
		err = a.Stop(ctx, id)
		assert.NoError(t, err)

		// cooldown
		time.Sleep(time.Millisecond * 100)

		// then all messahes should be delivered to the GRPC server
		ts.AssertMessagesProcessed(t, id, 4)
	})

	t.Run("cleaning GRPC connections in adapter on Stop", func(t *testing.T) {
		// given new test server
		ts := StartTestServer(t)

		ctx := context.Background()
		id1 := "id1"
		id2 := "id2"
		id3 := "id3"

		// and connection
		grpcConn, err := agentclient.NewGRPCConnection(ctx, true, true, ts.Url, "", log.DefaultLogger)
		if err != nil {
			t.Fatalf("Failed to create gRPC connection: %v", err)
		}
		t.Cleanup(func() {
			grpcConn.Close()
		})
		grpcClient := pb.NewCloudLogsServiceClient(grpcConn)
		a := NewCloudAdapter(grpcClient, "APIKEY")

		// when 3 streams are initialized, data is sent, and then stopped
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

		// then messages should be delivered
		ts.AssertMessagesProcessed(t, id1, 1)
		ts.AssertMessagesProcessed(t, id2, 1)
		ts.AssertMessagesProcessed(t, id3, 1)

		// and no stream are registered anymore in cloud adapter
		assertNoStreams(t, a)
	})

	t.Run("Send and receive a lot of messages", func(t *testing.T) {
		// given test server
		ts := StartTestServer(t)

		ctx := context.Background()
		id := "id1M"

		// and grpc connetion to the server
		grpcConn, err := agentclient.NewGRPCConnection(ctx, true, true, ts.Url, "", log.DefaultLogger)
		if err != nil {
			t.Fatalf("Failed to create gRPC connection: %v", err)
		}
		t.Cleanup(func() {
			grpcConn.Close()
		})

		// and logs stream client
		grpcClient := pb.NewCloudLogsServiceClient(grpcConn)
		a := NewCloudAdapter(grpcClient, "APIKEY")

		// when streams are initialized
		err = a.Init(ctx, id)
		assert.NoError(t, err)

		messageCount := 1000
		for i := 0; i < messageCount; i++ {
			// and data is sent
			err = a.Notify(ctx, id, *events.NewLog("log1"))
			assert.NoError(t, err)
		}

		// cooldown
		time.Sleep(time.Millisecond * 100)

		// then messages should be delivered to GRPC server
		ts.AssertMessagesProcessed(t, id, messageCount)
	})

	t.Run("Send to a lot of streams in parallel", func(t *testing.T) {
		// given test server
		ts := StartTestServer(t)

		ctx := context.Background()

		// and grpc connetion to the server
		grpcConn, err := agentclient.NewGRPCConnection(ctx, true, true, ts.Url, "", log.DefaultLogger)
		if err != nil {
			t.Fatalf("Failed to create gRPC connection: %v", err)
		}
		t.Cleanup(func() {
			grpcConn.Close()
		})

		// and logs stream client
		grpcClient := pb.NewCloudLogsServiceClient(grpcConn)
		a := NewCloudAdapter(grpcClient, "APIKEY")

		streamsCount := 10
		messageCount := 100

		// when streams are initialized
		var wg sync.WaitGroup
		wg.Add(streamsCount)
		for j := 0; j < streamsCount; j++ {
			err = a.Init(ctx, fmt.Sprintf("id%d", j))
			assert.NoError(t, err)

			go func(j int) {
				defer wg.Done()
				for i := 0; i < messageCount; i++ {
					// and when data are sent
					err = a.Notify(ctx, fmt.Sprintf("id%d", j), *events.NewLog("log1"))
					assert.NoError(t, err)
				}
			}(j)
		}

		wg.Wait()

		// and wait for cooldown
		time.Sleep(time.Millisecond * 100)

		// then each stream should receive valid data amount
		for j := 0; j < streamsCount; j++ {
			ts.AssertMessagesProcessed(t, fmt.Sprintf("id%d", j), messageCount)
		}
	})

}

func assertNoStreams(t *testing.T, a *CloudAdapter) {
	t.Helper()
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
		ready:    make(chan struct{}),
		errChan:  make(chan error, 1),
		shutdown: make(chan struct{}),
	}
}

type TestServer struct {
	Url string
	pb.UnimplementedCloudLogsServiceServer
	Received map[string][]*pb.Log
	lock     sync.Mutex
	ready    chan struct{}
	errChan  chan error
	shutdown chan struct{}
	server   *grpc.Server
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

	s.lock.Lock()
	s.Received[id] = []*pb.Log{}
	s.lock.Unlock()

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

		s.lock.Lock()
		s.Received[id] = append(s.Received[id], in)
		s.lock.Unlock()
	}
}

func (s *TestServer) WithRandomPort(t *testing.T) *TestServer {
	t.Helper()
	// Try up to 10 times to find an available port
	for i := 0; i < 10; i++ {
		port := rand.Intn(1000) + 18000
		s.Url = fmt.Sprintf("127.0.0.1:%d", port)

		// Check if port is available
		lis, err := net.Listen("tcp", s.Url)
		if err == nil {
			lis.Close()
			return s
		}
	}
	t.Fatal("Could not find available port after 10 attempts")
	return nil // unreachable, but makes compiler happy
}

func (s *TestServer) Run() error {
	lis, err := net.Listen("tcp", s.Url)
	if err != nil {
		err = errors.Wrapf(err, "failed to listen on %s", s.Url)
		s.errChan <- err
		close(s.ready) // Signal ready so WaitForReady doesn't block forever
		return err
	}

	var opts []grpc.ServerOption
	creds := insecure.NewCredentials()
	opts = append(opts, grpc.Creds(creds), grpc.MaxRecvMsgSize(math.MaxInt32))

	// register server logs
	s.server = grpc.NewServer(opts...)
	s.server.RegisterService(&pb.CloudLogsService_ServiceDesc, s)

	// Signal that server is ready to accept connections
	close(s.ready)

	// Start serving in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.Serve(lis); err != nil {
			errChan <- errors.Wrap(err, "grpc server error")
		}
	}()

	// Wait for shutdown signal or serve error
	select {
	case <-s.shutdown:
		s.server.GracefulStop()
		return nil
	case err := <-errChan:
		return err
	}
}

func (s *TestServer) AssertMessagesProcessed(t *testing.T, id string, messageCount int) {
	var received int

	for i := 0; i < 100; i++ {
		s.lock.Lock()
		received = len(s.Received[id])
		s.lock.Unlock()

		if received == messageCount {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}

	assert.Equal(t, messageCount, received)
}

func (s *TestServer) WaitForReady(t *testing.T, timeout time.Duration) {
	t.Helper()
	select {
	case <-s.ready:
		// Check if there was a startup error
		select {
		case err := <-s.errChan:
			t.Fatalf("Test server failed to start: %v", err)
		default:
			// No error, server is ready
		}
	case err := <-s.errChan:
		t.Fatalf("Test server failed to start: %v", err)
	case <-time.After(timeout):
		t.Fatal("Test server failed to start within timeout")
	}
}

func (s *TestServer) Shutdown() {
	close(s.shutdown)
}

// StartTestServer starts a test server and registers cleanup
func StartTestServer(t *testing.T) *TestServer {
	t.Helper()
	ts := NewTestServer().WithRandomPort(t)

	// Start server in background
	go func() {
		if err := ts.Run(); err != nil {
			t.Logf("Test server error: %v", err)
		}
	}()

	// Wait for server to be ready
	ts.WaitForReady(t, 5*time.Second)

	// Register cleanup
	t.Cleanup(func() {
		ts.Shutdown()
	})

	return ts
}
