package grpc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cloudflare/backoff"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

type fakeExecutionUpdatesClient struct {
	getExecutionUpdates func(context.Context, *executionv1.GetExecutionUpdatesRequest, ...grpc.CallOption) (*executionv1.GetExecutionUpdatesResponse, error)
}

func (f fakeExecutionUpdatesClient) GetExecutionUpdates(ctx context.Context, in *executionv1.GetExecutionUpdatesRequest, opts ...grpc.CallOption) (*executionv1.GetExecutionUpdatesResponse, error) {
	return f.getExecutionUpdates(ctx, in, opts...)
}

func (fakeExecutionUpdatesClient) SetExecutionScheduling(context.Context, *executionv1.SetExecutionSchedulingRequest, ...grpc.CallOption) (*executionv1.SetExecutionSchedulingResponse, error) {
	return nil, nil
}

func (fakeExecutionUpdatesClient) AcceptExecution(context.Context, *executionv1.AcceptExecutionRequest, ...grpc.CallOption) (*executionv1.AcceptExecutionResponse, error) {
	return nil, nil
}

func (fakeExecutionUpdatesClient) DeclineExecution(context.Context, *executionv1.DeclineExecutionRequest, ...grpc.CallOption) (*executionv1.DeclineExecutionResponse, error) {
	return nil, nil
}

func (fakeExecutionUpdatesClient) GetExecutionWorkflow(context.Context, *executionv1.GetExecutionWorkflowRequest, ...grpc.CallOption) (*executionv1.GetExecutionWorkflowResponse, error) {
	return nil, nil
}

type noopRunner struct{}

func (noopRunner) Execute(executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	return nil, nil
}
func (noopRunner) Pause(string) error  { return nil }
func (noopRunner) Resume(string) error { return nil }
func (noopRunner) Abort(string, string, string) error {
	return nil
}
func (noopRunner) Cancel(string, string, string) error {
	return nil
}

func TestClientStart_UsesSingleCappedBackoffDelayPerFailure(t *testing.T) {
	previousBackoff := newPollBackoff
	previousAfter := after
	t.Cleanup(func() {
		newPollBackoff = previousBackoff
		after = previousAfter
	})

	newPollBackoff = func(interval time.Duration) pollBackoff {
		return backoff.NewWithoutJitter(8*time.Millisecond, interval)
	}

	var mu sync.Mutex
	var delays []time.Duration
	after = func(delay time.Duration) <-chan time.Time {
		mu.Lock()
		delays = append(delays, delay)
		mu.Unlock()
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls int
	client := Client{
		client: fakeExecutionUpdatesClient{getExecutionUpdates: func(context.Context, *executionv1.GetExecutionUpdatesRequest, ...grpc.CallOption) (*executionv1.GetExecutionUpdatesResponse, error) {
			calls++
			if calls == 5 {
				cancel()
				return &executionv1.GetExecutionUpdatesResponse{}, nil
			}
			return nil, status.Error(codes.DeadlineExceeded, "context deadline exceeded")
		}},
		logger:       zap.NewNop().Sugar(),
		runner:       noopRunner{},
		callTimeout:  time.Second,
		pollInterval: time.Millisecond,
	}

	done := make(chan error, 1)
	go func() {
		done <- client.Start(ctx, "env-1")
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("client start did not stop in time")
	}

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []time.Duration{time.Millisecond, 2 * time.Millisecond, 4 * time.Millisecond, 8 * time.Millisecond}, delays)
}

func TestClientStart_ResetsBackoffAfterSuccessfulPoll(t *testing.T) {
	previousBackoff := newPollBackoff
	previousAfter := after
	t.Cleanup(func() {
		newPollBackoff = previousBackoff
		after = previousAfter
	})

	newPollBackoff = func(interval time.Duration) pollBackoff {
		return backoff.NewWithoutJitter(8*time.Millisecond, interval)
	}

	var mu sync.Mutex
	var delays []time.Duration
	after = func(delay time.Duration) <-chan time.Time {
		mu.Lock()
		delays = append(delays, delay)
		mu.Unlock()
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls int
	client := Client{
		client: fakeExecutionUpdatesClient{getExecutionUpdates: func(context.Context, *executionv1.GetExecutionUpdatesRequest, ...grpc.CallOption) (*executionv1.GetExecutionUpdatesResponse, error) {
			calls++
			switch calls {
			case 1, 2, 4, 5:
				return nil, status.Error(codes.DeadlineExceeded, "context deadline exceeded")
			case 3:
				return &executionv1.GetExecutionUpdatesResponse{}, nil
			case 6:
				cancel()
				return &executionv1.GetExecutionUpdatesResponse{}, nil
			default:
				return &executionv1.GetExecutionUpdatesResponse{}, nil
			}
		}},
		logger:       zap.NewNop().Sugar(),
		runner:       noopRunner{},
		callTimeout:  time.Second,
		pollInterval: time.Millisecond,
	}

	done := make(chan error, 1)
	go func() {
		done <- client.Start(ctx, "env-1")
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("client start did not stop in time")
	}

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []time.Duration{time.Millisecond, 2 * time.Millisecond, time.Millisecond, 2 * time.Millisecond}, delays)
}
