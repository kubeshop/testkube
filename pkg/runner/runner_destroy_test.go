package runner

import (
	"context"
	stderrors "errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

// A cluster hiccup during cleanup fails Destroy for a few attempts, then it succeeds.
// The retry helper hides the transient failures from the caller.
func TestDestroyResources_RetriesTransientFailuresThenSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const transientFailures = 2
	var calls int32
	worker := executionworkertypes.NewMockWorker(ctrl)
	worker.EXPECT().
		Destroy(gomock.Any(), "exec-1", gomock.Any()).
		DoAndReturn(func(context.Context, string, executionworkertypes.DestroyOptions) error {
			if atomic.AddInt32(&calls, 1) <= transientFailures {
				return stderrors.New("kube-apiserver unavailable")
			}
			return nil
		}).
		Times(transientFailures + 1)

	r := &runner{worker: worker, cleanupRetryDelay: time.Millisecond}

	err := r.destroyResources(context.Background(), "exec-1")

	assert.NoError(t, err)
}

// The k8s API stays unreachable for the whole retry budget; the helper eventually
// gives up and surfaces the last error so the caller can log it.
func TestDestroyResources_GivesUpAfterRetryBudget(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	wantErr := stderrors.New("kube-apiserver unavailable")
	worker := executionworkertypes.NewMockWorker(ctrl)
	worker.EXPECT().
		Destroy(gomock.Any(), "exec-1", gomock.Any()).
		Return(wantErr).
		Times(CleanupResourcesRetryCount)

	r := &runner{worker: worker, cleanupRetryDelay: time.Millisecond}

	err := r.destroyResources(context.Background(), "exec-1")

	assert.ErrorIs(t, err, wantErr)
}
