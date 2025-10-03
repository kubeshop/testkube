package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

// Simple mock event emitter
type simpleMockEventEmitter struct{}

func (m *simpleMockEventEmitter) Notify(event testkube.Event) {}

func TestAgentLoop_Start_SimpleReconnectionDelay(t *testing.T) {
	// Create test gRPC server and client
	testServer := &testGRPCServer{
		getRunnerRequestsErrorToReturn:                   status.Error(codes.Unavailable, "server unavailable"),
		getRunnerRequestSendPing:                         true,
		getTestWorkflowNotificationsSendPing:             true,
		getTestWorkflowServiceNotificationsSendPing:      true,
		getTestWorkflowParallelStepNotificationsSendPing: true,
	}
	grpcServer, conn, testServer := createTestGRPCServer(testServer)
	defer grpcServer.Stop()
	defer conn.Close()

	// Create real gRPC client
	grpcClient := cloud.NewTestKubeCloudAPIClient(conn)

	// Create control plane client
	mockClient := controlplaneclient.New(grpcClient, config.ProContext{
		NewArchitecture: true,
		Agent: config.ProContextAgent{
			ID: "test-agent",
		},
	}, controlplaneclient.ClientOptions{
		StorageSkipVerify: true,
		Runtime: controlplaneclient.RuntimeConfig{
			Namespace: "test-namespace",
		},
		SendTimeout: 5 * time.Second,
		RecvTimeout: 5 * time.Second,
	}, zap.NewNop().Sugar())

	// Create mocks for other dependencies
	mockRunner := &MockRunner{}
	mockWorker := executionworkertypes.NewMockWorker(nil)
	mockEmitter := &simpleMockEventEmitter{}

	logger := zap.NewExample().Sugar()

	// Create agent loop
	agent := newAgentLoop(
		mockRunner,
		mockWorker,
		logger,
		mockEmitter,
		mockClient,
		testworkflowconfig.ControlPlaneConfig{},
		config.ProContext{
			NewArchitecture: true,
			Agent: config.ProContextAgent{
				ID: "test-agent",
			},
		},
		"test-org",
		"test-env",
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the agent loop
	err := agent.Start(ctx, true)

	// Should return context deadline exceeded error (expected) and should have made multiple calls
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.GreaterOrEqual(t, testServer.GetCallCount(), 2)
}

func TestAgentLoop_GetRunnerRequests_ReconnectionOnReceiveTimeout(t *testing.T) {
	testServer := &testGRPCServer{
		getRunnerRequestSendPing:                         false,
		getTestWorkflowNotificationsSendPing:             true,
		getTestWorkflowServiceNotificationsSendPing:      true,
		getTestWorkflowParallelStepNotificationsSendPing: true,
	}
	grpcServer, conn, testServer := createTestGRPCServer(testServer)
	defer grpcServer.Stop()
	defer conn.Close()

	grpcClient := cloud.NewTestKubeCloudAPIClient(conn)

	testTimeout := time.Second

	mockClient := controlplaneclient.New(grpcClient, config.ProContext{
		NewArchitecture: true,
		Agent: config.ProContextAgent{
			ID: "test-agent",
		},
	}, controlplaneclient.ClientOptions{
		StorageSkipVerify: true,
		Runtime: controlplaneclient.RuntimeConfig{
			Namespace: "test-namespace",
		},
		SendTimeout: 100 * time.Second,
		RecvTimeout: testTimeout,
	}, zap.NewExample().Sugar())

	// Create mocks for other dependencies
	mockRunner := &MockRunner{}
	mockWorker := executionworkertypes.NewMockWorker(nil)
	mockEmitter := &simpleMockEventEmitter{}

	// Enable debug logging to see what's happening
	logger := zap.NewExample().Sugar()

	// Create agent loop
	agent := newAgentLoop(
		mockRunner,
		mockWorker,
		logger,
		mockEmitter,
		mockClient,
		testworkflowconfig.ControlPlaneConfig{},
		config.ProContext{
			NewArchitecture: true,
			Agent: config.ProContextAgent{
				ID: "test-agent",
			},
		},
		"test-org",
		"test-env",
	)

	// Create context with timeout longer than 2x receive timeout + reconnect timeout
	ctx, cancel := context.WithTimeout(context.Background(), (2*testTimeout)+agentLoopReconnectionDelay)
	defer cancel()

	// Start the agent loop
	err := agent.Start(ctx, true)

	// Should return context deadline exceeded error (expected) and should have made multiple calls
	// due to receive timeout causing reconnections
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")

	// Should have made multiple calls due to receive timeout reconnections
	assert.GreaterOrEqual(t, testServer.GetCallCount(), 2,
		"Expected at least 2 calls due to receive timeout reconnections, got %d", testServer.GetCallCount())
}

func TestAgentLoop_GetNotifications_ReconnectionOnReceiveTimeout(t *testing.T) {
	testServer := &testGRPCServer{
		getRunnerRequestSendPing:                         true,
		getTestWorkflowNotificationsSendPing:             false,
		getTestWorkflowServiceNotificationsSendPing:      true,
		getTestWorkflowParallelStepNotificationsSendPing: true,
	}
	grpcServer, conn, testServer := createTestGRPCServer(testServer)
	defer grpcServer.Stop()
	defer conn.Close()

	grpcClient := cloud.NewTestKubeCloudAPIClient(conn)

	testTimeout := time.Second

	mockClient := controlplaneclient.New(grpcClient, config.ProContext{
		NewArchitecture: true,
		Agent: config.ProContextAgent{
			ID: "test-agent",
		},
	}, controlplaneclient.ClientOptions{
		StorageSkipVerify: true,
		Runtime: controlplaneclient.RuntimeConfig{
			Namespace: "test-namespace",
		},
		SendTimeout: 100 * time.Second,
		RecvTimeout: testTimeout,
	}, zap.NewExample().Sugar())

	// Create mocks for other dependencies
	mockRunner := &MockRunner{}
	mockWorker := executionworkertypes.NewMockWorker(nil)
	mockEmitter := &simpleMockEventEmitter{}

	// Enable debug logging to see what's happening
	logger := zap.NewExample().Sugar()

	// Create agent loop
	agent := newAgentLoop(
		mockRunner,
		mockWorker,
		logger,
		mockEmitter,
		mockClient,
		testworkflowconfig.ControlPlaneConfig{},
		config.ProContext{
			NewArchitecture: true,
			Agent: config.ProContextAgent{
				ID: "test-agent",
			},
		},
		"test-org",
		"test-env",
	)

	// Create context with timeout longer than 2x receive timeout + reconnect timeout
	ctx, cancel := context.WithTimeout(context.Background(), (2*testTimeout)+agentLoopReconnectionDelay)
	defer cancel()

	// Start the agent loop
	err := agent.Start(ctx, true)

	// Should return context deadline exceeded error (expected) and should have made multiple calls
	// due to receive timeout causing reconnections
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")

	// Should have made multiple calls due to receive timeout reconnections
	// Each call should timeout after 2 seconds, so we expect at least 2 calls in 7 seconds
	assert.GreaterOrEqual(t, testServer.GetCallCount(), 2,
		"Expected at least 2 calls due to receive timeout reconnections, got %d", testServer.GetCallCount())
}
