package grpc

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

// stopRecorder records the actor and the reason each stop request carries.
type stopRecorder struct {
	mu       sync.Mutex
	aborted  map[string]stop
	canceled map[string]stop
}

type stop struct {
	actor  string
	reason string
}

func (r *stopRecorder) Execute(executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	return nil, nil
}
func (r *stopRecorder) Pause(string) error  { return nil }
func (r *stopRecorder) Resume(string) error { return nil }
func (r *stopRecorder) Abort(id, actor, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.aborted[id] = stop{actor: actor, reason: reason}
	return nil
}
func (r *stopRecorder) Cancel(id, actor, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.canceled[id] = stop{actor: actor, reason: reason}
	return nil
}

func TestClient_executeResponse_PassesStopActorAndReason(t *testing.T) {
	tests := []struct {
		name         string
		to           executionv1.ExecutionState
		actor        *string
		reason       *string
		wantAborted  map[string]stop
		wantCanceled map[string]stop
	}{
		{
			name:         "abort with a cause",
			to:           executionv1.ExecutionState_EXECUTION_STATE_ABORTED,
			reason:       proto.String("execution-timeout"),
			wantAborted:  map[string]stop{"exec-1": {reason: "execution-timeout"}},
			wantCanceled: map[string]stop{},
		},
		{
			name:         "cancel with an actor and a cause",
			to:           executionv1.ExecutionState_EXECUTION_STATE_CANCELLED,
			actor:        proto.String("quality-loop"),
			reason:       proto.String("superseded"),
			wantAborted:  map[string]stop{},
			wantCanceled: map[string]stop{"exec-1": {actor: "quality-loop", reason: "superseded"}},
		},
		{
			name:         "cancel from a control plane that sends no actor and no reason",
			to:           executionv1.ExecutionState_EXECUTION_STATE_CANCELLED,
			wantAborted:  map[string]stop{},
			wantCanceled: map[string]stop{"exec-1": {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &stopRecorder{aborted: map[string]stop{}, canceled: map[string]stop{}}
			c := Client{runner: recorder, logger: zap.NewNop().Sugar()}

			c.executeResponse(context.Background(), &executionv1.GetExecutionUpdatesResponse{
				Update: []*executionv1.ExecutionStateTransition{{
					ExecutionId:  proto.String("exec-1"),
					TransitionTo: tt.to.Enum(),
					Actor:        tt.actor,
					Reason:       tt.reason,
				}},
			})

			require.Equal(t, tt.wantAborted, recorder.aborted)
			require.Equal(t, tt.wantCanceled, recorder.canceled)
		})
	}
}
