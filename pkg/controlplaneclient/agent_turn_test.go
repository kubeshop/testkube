package controlplaneclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
)

// agentTurnServer records each agent turn call. It fails the first calls with the
// errors in fail, in order, then answers with the turn.
type agentTurnServer struct {
	cloud.TestKubeCloudAPIClient
	fail     []error
	requests []proto.Message
	metadata []metadata.MD
}

func (s *agentTurnServer) record(ctx context.Context, req proto.Message) (*cloud.AgentTurn, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	s.requests = append(s.requests, req)
	s.metadata = append(s.metadata, md)
	if len(s.fail) > 0 {
		err := s.fail[0]
		s.fail = s.fail[1:]
		return nil, err
	}
	return &cloud.AgentTurn{Id: "turn-1"}, nil
}

func (s *agentTurnServer) StartAgentTurn(ctx context.Context, req *cloud.StartAgentTurnRequest, _ ...grpc.CallOption) (*cloud.AgentTurn, error) {
	return s.record(ctx, req)
}

func (s *agentTurnServer) GetAgentTurn(ctx context.Context, req *cloud.GetAgentTurnRequest, _ ...grpc.CallOption) (*cloud.AgentTurn, error) {
	return s.record(ctx, req)
}

func (s *agentTurnServer) CancelAgentTurn(ctx context.Context, req *cloud.CancelAgentTurnRequest, _ ...grpc.CallOption) (*cloud.AgentTurn, error) {
	return s.record(ctx, req)
}

func TestClient_AgentTurn(t *testing.T) {
	withShortCallBackoff(t)

	start := &cloud.StartAgentTurnRequest{RequestId: "req-1", StepRef: "ref-1", AgentName: "triage", Prompt: "Find the cause."}
	unavailable := status.Error(codes.Unavailable, "no healthy upstream")

	tests := []struct {
		name string
		fail []error
		call func(c Client) (*cloud.AgentTurn, error)
		// wantRequests is each request that reaches the server, in order.
		wantRequests []proto.Message
	}{
		{
			name:         "start sends the request as it is",
			call:         func(c Client) (*cloud.AgentTurn, error) { return c.StartAgentTurn(context.Background(), start) },
			wantRequests: []proto.Message{start},
		},
		{
			name:         "start sends the same request ID again after a transient error",
			fail:         []error{unavailable},
			call:         func(c Client) (*cloud.AgentTurn, error) { return c.StartAgentTurn(context.Background(), start) },
			wantRequests: []proto.Message{start, start},
		},
		{
			name:         "get sends the turn and the cursor",
			call:         func(c Client) (*cloud.AgentTurn, error) { return c.GetAgentTurn(context.Background(), "turn-1", 7) },
			wantRequests: []proto.Message{&cloud.GetAgentTurnRequest{TurnId: "turn-1", After: 7}},
		},
		{
			name:         "cancel sends the turn",
			call:         func(c Client) (*cloud.AgentTurn, error) { return c.CancelAgentTurn(context.Background(), "turn-1") },
			wantRequests: []proto.Message{&cloud.CancelAgentTurnRequest{TurnId: "turn-1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &agentTurnServer{fail: tt.fail}
			c := &client{
				client:     server,
				proContext: config.ProContext{APIKey: "execution-token", OrgID: "org-1", EnvID: "env-1"},
				opts:       ClientOptions{ExecutionID: "exec-1"},
			}

			turn, err := tt.call(c)

			require.NoError(t, err)
			assert.Equal(t, "turn-1", turn.GetId())
			require.Len(t, server.requests, len(tt.wantRequests))
			for i, want := range tt.wantRequests {
				assert.Truef(t, proto.Equal(want, server.requests[i]), "request %d: want %v, got %v", i, want, server.requests[i])
				assert.Equal(t, []string{"execution-token"}, server.metadata[i].Get(AgentSecretKeyMetadataName))
				assert.Equal(t, []string{"exec-1"}, server.metadata[i].Get(ExecutionIdMetadataName))
			}
		})
	}
}
