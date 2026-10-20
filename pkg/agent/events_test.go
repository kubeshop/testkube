package agent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
)

// sendOnlyClient implements just the Send stream of the Control Plane client. Any other call
// panics on the nil embedded interface, which is what proves the health check loop makes none.
type sendOnlyClient struct {
	cloud.TestKubeCloudAPIClient
	sent chan *cloud.WebsocketData
}

func (c *sendOnlyClient) Send(_ context.Context, _ ...grpc.CallOption) (cloud.TestKubeCloudAPI_SendClient, error) {
	return &recordingSendStream{sent: c.sent}, nil
}

type recordingSendStream struct {
	grpc.ClientStream
	sent chan *cloud.WebsocketData
}

func (s *recordingSendStream) Send(msg *cloud.WebsocketData) error {
	s.sent <- msg
	return nil
}

func (s *recordingSendStream) CloseAndRecv() (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// The loop is exercised directly: RunHealthcheck only adds runWithReconnect, whose reconnection
// delay would hold the test for seconds after cancellation.
func TestEventLoop_SendsHealthChecksWithoutEvents(t *testing.T) {
	client := &sendOnlyClient{sent: make(chan *cloud.WebsocketData, 10)}
	ag, err := NewAgent(zap.NewNop().Sugar(), nil, client, "cluster", "cluster", &config.ProContext{}, "", nil)
	require.NoError(t, err)
	ag.healthInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- ag.runEventLoop(ctx) }()

	for i := 0; i < 2; i++ {
		select {
		case msg := <-client.sent:
			assert.Equal(t, cloud.Opcode_HEALTH_CHECK, msg.Opcode)
			assert.Empty(t, msg.Body)
		case <-time.After(5 * time.Second):
			t.Fatal("no health check sent")
		}
	}

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("health check loop did not stop on cancellation")
	}
}
