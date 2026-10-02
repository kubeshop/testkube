package runner

import (
	"context"
	stderrors "errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

// abortExecution runs after Monitor has already given up. A transient failure
// on any of its control-plane calls without retry would leave the execution
// stuck in running state. The retry loop must hide the transient failures
// from the caller.
func TestAbortExecution_RetriesTransientFailures(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const transientFailures = 2
	transient := stderrors.New("control-plane unavailable")

	client := controlplaneclient.NewMockClient(ctrl)

	var getCalls int32
	client.EXPECT().
		GetExecution(gomock.Any(), "env-1", "exec-1").
		DoAndReturn(func(context.Context, string, string) (*testkube.TestWorkflowExecution, error) {
			if atomic.AddInt32(&getCalls, 1) <= transientFailures {
				return nil, transient
			}
			return &testkube.TestWorkflowExecution{
				Id:     "exec-1",
				Result: &testkube.TestWorkflowResult{},
			}, nil
		}).
		Times(transientFailures + 1)

	var updateCalls int32
	client.EXPECT().
		UpdateExecutionResult(gomock.Any(), "env-1", "exec-1", gomock.Any()).
		DoAndReturn(func(context.Context, string, string, *testkube.TestWorkflowResult) error {
			if atomic.AddInt32(&updateCalls, 1) <= transientFailures {
				return transient
			}
			return nil
		}).
		Times(transientFailures + 1)

	var finishCalls int32
	client.EXPECT().
		FinishExecutionResult(gomock.Any(), "env-1", "exec-1", gomock.Any()).
		DoAndReturn(func(context.Context, string, string, *testkube.TestWorkflowResult) error {
			if atomic.AddInt32(&finishCalls, 1) <= transientFailures {
				return transient
			}
			return nil
		}).
		Times(transientFailures + 1)

	worker := executionworkertypes.NewMockWorker(ctrl)
	var abortCalls int32
	worker.EXPECT().
		Abort(gomock.Any(), "exec-1", gomock.Any()).
		DoAndReturn(func(context.Context, string, executionworkertypes.DestroyOptions) error {
			if atomic.AddInt32(&abortCalls, 1) <= transientFailures {
				return transient
			}
			return nil
		}).
		Times(transientFailures + 1)

	r := &runner{worker: worker, client: client, abortRetryDelay: time.Millisecond}

	err := r.abortExecution(context.Background(), "env-1", "exec-1")
	assert.NoError(t, err)
}

// If every attempt of the first call keeps failing, the whole abort chain
// surfaces the error so the caller can alert on-call.
func TestAbortExecution_GivesUpAfterRetryBudget(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transient := stderrors.New("control-plane unavailable")

	client := controlplaneclient.NewMockClient(ctrl)
	client.EXPECT().
		GetExecution(gomock.Any(), "env-1", "exec-1").
		Return(nil, transient).
		Times(AbortExecutionRetryCount)

	worker := executionworkertypes.NewMockWorker(ctrl)

	r := &runner{worker: worker, client: client, abortRetryDelay: time.Millisecond}

	err := r.abortExecution(context.Background(), "env-1", "exec-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get execution")
}
