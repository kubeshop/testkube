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

// stopRecorder records the reason each stop request carries.
type stopRecorder struct {
	mu       sync.Mutex
	aborted  map[string]string
	canceled map[string]string
}

func (r *stopRecorder) Execute(executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	return nil, nil
}
func (r *stopRecorder) Pause(string) error  { return nil }
func (r *stopRecorder) Resume(string) error { return nil }
func (r *stopRecorder) Abort(id, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.aborted[id] = reason
	return nil
}
func (r *stopRecorder) Cancel(id, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.canceled[id] = reason
	return nil
}

func TestClient_executeResponse_PassesStopReason(t *testing.T) {
	tests := []struct {
		name         string
		to           executionv1.ExecutionState
		reason       *string
		wantAborted  map[string]string
		wantCanceled map[string]string
	}{
		{
			name:         "abort with a cause",
			to:           executionv1.ExecutionState_EXECUTION_STATE_ABORTED,
			reason:       proto.String("the execution ran for too long"),
			wantAborted:  map[string]string{"exec-1": "the execution ran for too long"},
			wantCanceled: map[string]string{},
		},
		{
			name:         "cancel with a cause",
			to:           executionv1.ExecutionState_EXECUTION_STATE_CANCELLED,
			reason:       proto.String("all executions of the workflow were canceled"),
			wantAborted:  map[string]string{},
			wantCanceled: map[string]string{"exec-1": "all executions of the workflow were canceled"},
		},
		{
			name:         "cancel from a control plane that sends no reason",
			to:           executionv1.ExecutionState_EXECUTION_STATE_CANCELLED,
			reason:       nil,
			wantAborted:  map[string]string{},
			wantCanceled: map[string]string{"exec-1": ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &stopRecorder{aborted: map[string]string{}, canceled: map[string]string{}}
			c := Client{runner: recorder, logger: zap.NewNop().Sugar()}

			c.executeResponse(context.Background(), &executionv1.GetExecutionUpdatesResponse{
				Update: []*executionv1.ExecutionStateTransition{{
					ExecutionId:  proto.String("exec-1"),
					TransitionTo: tt.to.Enum(),
					Reason:       tt.reason,
				}},
			})

			require.Equal(t, tt.wantAborted, recorder.aborted)
			require.Equal(t, tt.wantCanceled, recorder.canceled)
		})
	}
}
