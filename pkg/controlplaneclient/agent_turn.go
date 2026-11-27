package controlplaneclient

import (
	"context"

	"github.com/kubeshop/testkube/pkg/cloud"
)

// AgentTurnClient starts and follows one turn of an AI Agent for a step of an execution.
//
// The control plane takes the organization, the environment and the execution from the
// execution token, so the methods have no parameters for them.
//
// Each method retries a transient error. A retry of StartAgentTurn is safe because the
// request ID identifies the turn, and the two other calls change nothing on a repeat.
type AgentTurnClient interface {
	// StartAgentTurn starts a turn. A call with a request ID that the control plane knows
	// returns the turn of the first call.
	StartAgentTurn(ctx context.Context, req *cloud.StartAgentTurnRequest) (*cloud.AgentTurn, error)
	// GetAgentTurn returns the state of the turn and its events after the cursor.
	GetAgentTurn(ctx context.Context, turnId string, after uint32) (*cloud.AgentTurn, error)
	// CancelAgentTurn asks the control plane to stop the turn.
	CancelAgentTurn(ctx context.Context, turnId string) (*cloud.AgentTurn, error)
}

func (c *client) StartAgentTurn(ctx context.Context, req *cloud.StartAgentTurnRequest) (*cloud.AgentTurn, error) {
	return callWithRetry(ctx, c.metadata().GRPC(), c.client.StartAgentTurn, req)
}

func (c *client) GetAgentTurn(ctx context.Context, turnId string, after uint32) (*cloud.AgentTurn, error) {
	return callWithRetry(ctx, c.metadata().GRPC(), c.client.GetAgentTurn, &cloud.GetAgentTurnRequest{TurnId: turnId, After: after})
}

func (c *client) CancelAgentTurn(ctx context.Context, turnId string) (*cloud.AgentTurn, error) {
	return callWithRetry(ctx, c.metadata().GRPC(), c.client.CancelAgentTurn, &cloud.CancelAgentTurnRequest{TurnId: turnId})
}
