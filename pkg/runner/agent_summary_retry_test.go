package runner

import (
	"context"
	stderrors "errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

func newSummaryRetryAgentLoop(worker executionworkertypes.Worker) *agentLoop {
	return &agentLoop{
		worker:            worker,
		logger:            zap.NewNop().Sugar(),
		summaryRetryDelay: time.Millisecond,
	}
}

// Summary times out a few times, then succeeds; the helper retries and returns that result.
func TestSummaryWithJobRetry_RetriesOnJobTimeoutThenSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	want := &executionworkertypes.SummaryResult{}
	const timeouts = 3
	var calls int32
	worker := executionworkertypes.NewMockWorker(ctrl)
	worker.EXPECT().
		Summary(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ executionworkertypes.GetOptions) (*executionworkertypes.SummaryResult, error) {
			if atomic.AddInt32(&calls, 1) <= timeouts {
				return nil, controller.ErrJobTimeout
			}
			return want, nil
		}).
		Times(timeouts + 1) // pins the retry count: stops on the first success

	a := newSummaryRetryAgentLoop(worker)

	status, err := a.summaryWithJobRetry(context.Background(), "exec-1", executionworkertypes.GetOptions{})

	assert.NoError(t, err)
	assert.Same(t, want, status)
}

// Errors other than ErrJobTimeout are not retried; they come back on the first call.
func TestSummaryWithJobRetry_NonTimeoutErrorReturnsImmediately(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	wantErr := stderrors.New("permanent failure")
	worker := executionworkertypes.NewMockWorker(ctrl)
	worker.EXPECT().
		Summary(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, wantErr).
		Times(1)

	a := newSummaryRetryAgentLoop(worker)

	status, err := a.summaryWithJobRetry(context.Background(), "exec-1", executionworkertypes.GetOptions{})

	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, status)
}

// A Job that never becomes visible is retried up to the budget, then the helper gives up and
// returns the last ErrJobTimeout. Times() pins the bound so the loop cannot run forever.
func TestSummaryWithJobRetry_GivesUpAfterRetryBudget(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	worker := executionworkertypes.NewMockWorker(ctrl)
	worker.EXPECT().
		Summary(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, controller.ErrJobTimeout).
		Times(GetNotificationsRetryCount)

	a := newSummaryRetryAgentLoop(worker)

	status, err := a.summaryWithJobRetry(context.Background(), "exec-1", executionworkertypes.GetOptions{})

	assert.ErrorIs(t, err, controller.ErrJobTimeout)
	assert.Nil(t, status)
}

// Cancelling the context interrupts the wait between attempts at once instead of
// using the full delay budget.
func TestSummaryWithJobRetry_ContextCancelledReturnsPromptly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	worker := executionworkertypes.NewMockWorker(ctrl)
	worker.EXPECT().
		Summary(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, string, executionworkertypes.GetOptions) (*executionworkertypes.SummaryResult, error) {
			cancel() // cancel mid-flight so the wait before the next attempt is cut short
			return nil, controller.ErrJobTimeout
		}).
		Times(1)

	a := newSummaryRetryAgentLoop(worker)
	a.summaryRetryDelay = time.Hour // only ctx cancellation can break this wait

	start := time.Now()
	status, err := a.summaryWithJobRetry(ctx, "exec-1", executionworkertypes.GetOptions{})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, status)
	assert.Less(t, time.Since(start), time.Second)
}

// Same as the retry test, but Summary calls the real controller.New. Against a fake cluster with
// no Job it returns the genuine controller.ErrJobTimeout, which also proves a not-yet-visible Job
// surfaces as that error (what the mock-based tests assume).
func TestSummaryWithJobRetry_RealControllerErrJobTimeoutThenRecovers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	clientSet := fake.NewSimpleClientset()
	const namespace, executionID = "tk-int", "exec-int"

	_, err := controller.New(context.Background(), clientSet, namespace, executionID, time.Now())
	assert.ErrorIs(t, err, controller.ErrJobTimeout)

	want := &executionworkertypes.SummaryResult{}
	const timeouts = 2
	var calls int32
	worker := executionworkertypes.NewMockWorker(ctrl)
	worker.EXPECT().
		Summary(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, id string, _ executionworkertypes.GetOptions) (*executionworkertypes.SummaryResult, error) {
			if atomic.AddInt32(&calls, 1) <= timeouts {
				_, err := controller.New(ctx, clientSet, namespace, id, time.Now())
				return nil, err
			}
			return want, nil
		}).
		Times(timeouts + 1)

	a := newSummaryRetryAgentLoop(worker)

	status, err := a.summaryWithJobRetry(context.Background(), executionID, executionworkertypes.GetOptions{})

	assert.NoError(t, err)
	assert.Same(t, want, status)
}
