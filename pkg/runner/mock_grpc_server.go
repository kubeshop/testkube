//nolint:staticcheck
package runner

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/kubeshop/testkube/pkg/cloud"
)

// Test gRPC server implementation
type testGRPCServer struct {
	cloud.UnimplementedTestKubeCloudAPIServer
	getRunnerRequestsErrorToReturn                        error
	getRunnerRequestsSendData                             *cloud.RunnerRequest
	getTestWorkflowNotificationsErrorToReturn             error
	getTestWorkflowServiceNotificationsErrorToReturn      error
	getTestWorkflowParallelStepNotificationsErrorToReturn error
	getRunnerRequestSendPing                              bool
	getTestWorkflowNotificationsSendPing                  bool
	getTestWorkflowServiceNotificationsSendPing           bool
	getTestWorkflowParallelStepNotificationsSendPing      bool
	callCount                                             int
	mu                                                    sync.Mutex
}

func (s *testGRPCServer) GetRunnerRequests(stream cloud.TestKubeCloudAPI_GetRunnerRequestsServer) error {
	s.mu.Lock()
	s.callCount++
	currentCall := s.callCount
	s.mu.Unlock()

	fmt.Printf("DEBUG: GetRunnerRequests called (call #%d)\n", currentCall)
	if s.getRunnerRequestsErrorToReturn != nil {
		fmt.Printf("DEBUG: GetRunnerRequests returning error: %v\n", s.getRunnerRequestsErrorToReturn)
		return s.getRunnerRequestsErrorToReturn
	}

	for {
		select {
		case <-stream.Context().Done():
			fmt.Printf("DEBUG: GetRunnerRequests context cancelled\n")
			return stream.Context().Err()
		case <-time.After(1 * time.Second):
			if s.getRunnerRequestSendPing {
				stream.Send(&cloud.RunnerRequest{
					Type: cloud.RunnerRequestType_PING,
				})
				var req cloud.RunnerRequest
				stream.RecvMsg(&req)
			}

			if s.getRunnerRequestsSendData != nil {
				stream.Send(s.getRunnerRequestsSendData)
			}
		}

	}

}

func (s *testGRPCServer) GetTestWorkflowNotificationsStream(stream cloud.TestKubeCloudAPI_GetTestWorkflowNotificationsStreamServer) error {
	if s.getTestWorkflowNotificationsErrorToReturn != nil {
		return s.getTestWorkflowNotificationsErrorToReturn
	}

	// Send an empty request and wait for context cancellation
	for {
		select {
		case <-stream.Context().Done():
			fmt.Printf("DEBUG: GetTestWorkflowNotificationsStream context cancelled\n")
			return stream.Context().Err()
		case <-time.After(1 * time.Second):
			if s.getTestWorkflowNotificationsSendPing {
				stream.Send(&cloud.TestWorkflowNotificationsRequest{
					StreamId:    "test-stream-id",
					RequestType: cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK,
				})
				var req cloud.TestWorkflowNotificationsRequest
				stream.RecvMsg(&req)
			}
		}
	}
}

func (s *testGRPCServer) GetTestWorkflowServiceNotificationsStream(stream cloud.TestKubeCloudAPI_GetTestWorkflowServiceNotificationsStreamServer) error {
	if s.getTestWorkflowServiceNotificationsErrorToReturn != nil {
		return s.getTestWorkflowServiceNotificationsErrorToReturn
	}

	// Send an empty request and wait for context cancellation
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-time.After(1 * time.Second):
			if s.getTestWorkflowServiceNotificationsSendPing {
				stream.Send(&cloud.TestWorkflowServiceNotificationsRequest{
					StreamId:    "test-stream-id",
					RequestType: cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK,
				})
				var req cloud.TestWorkflowServiceNotificationsRequest
				stream.RecvMsg(&req)
			}
		}
	}
}

func (s *testGRPCServer) GetTestWorkflowParallelStepNotificationsStream(stream cloud.TestKubeCloudAPI_GetTestWorkflowParallelStepNotificationsStreamServer) error {
	if s.getTestWorkflowParallelStepNotificationsErrorToReturn != nil {
		return s.getTestWorkflowParallelStepNotificationsErrorToReturn
	}

	// Send an empty request and wait for context cancellation
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-time.After(1 * time.Second):
			if s.getTestWorkflowParallelStepNotificationsSendPing {
				stream.Send(&cloud.TestWorkflowParallelStepNotificationsRequest{
					StreamId:    "test-stream-id",
					RequestType: cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK,
				})
				var req cloud.TestWorkflowParallelStepNotificationsRequest
				stream.RecvMsg(&req)
			}
		}
	}
}

func (s *testGRPCServer) GetCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.callCount
}

// Helper function to create a test gRPC server and client
func createTestGRPCServer(testServer *testGRPCServer) (*grpc.Server, *grpc.ClientConn, *testGRPCServer) {
	// Create a buffer-based listener for testing
	listener := bufconn.Listen(1024 * 1024)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	cloud.RegisterTestKubeCloudAPIServer(grpcServer, testServer)

	// Start server in background
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			// Ignore server closed errors
			fmt.Printf("DEBUG: grpcServer.Serve error: %v\n", err)
		}
	}()

	// Create client connection
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", //nolint
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}

	return grpcServer, conn, testServer
}
