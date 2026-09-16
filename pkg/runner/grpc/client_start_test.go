package grpc

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudflare/backoff"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
	testworkflowv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

type fakeExecutionUpdatesClient struct {
	getExecutionUpdates  func(context.Context, *executionv1.GetExecutionUpdatesRequest, ...grpc.CallOption) (*executionv1.GetExecutionUpdatesResponse, error)
	getExecutionWorkflow func(context.Context, *executionv1.GetExecutionWorkflowRequest, ...grpc.CallOption) (*executionv1.GetExecutionWorkflowResponse, error)
	acceptExecution      func(context.Context, *executionv1.AcceptExecutionRequest, ...grpc.CallOption) (*executionv1.AcceptExecutionResponse, error)
}

func (f fakeExecutionUpdatesClient) GetExecutionUpdates(ctx context.Context, in *executionv1.GetExecutionUpdatesRequest, opts ...grpc.CallOption) (*executionv1.GetExecutionUpdatesResponse, error) {
	return f.getExecutionUpdates(ctx, in, opts...)
}

func (fakeExecutionUpdatesClient) SetExecutionScheduling(context.Context, *executionv1.SetExecutionSchedulingRequest, ...grpc.CallOption) (*executionv1.SetExecutionSchedulingResponse, error) {
	return nil, nil
}

func (f fakeExecutionUpdatesClient) AcceptExecution(ctx context.Context, req *executionv1.AcceptExecutionRequest, opts ...grpc.CallOption) (*executionv1.AcceptExecutionResponse, error) {
	if f.acceptExecution == nil {
		return &executionv1.AcceptExecutionResponse{}, nil
	}
	return f.acceptExecution(ctx, req, opts...)
}

func (fakeExecutionUpdatesClient) DeclineExecution(context.Context, *executionv1.DeclineExecutionRequest, ...grpc.CallOption) (*executionv1.DeclineExecutionResponse, error) {
	return nil, nil
}

func (f fakeExecutionUpdatesClient) GetExecutionWorkflow(ctx context.Context, req *executionv1.GetExecutionWorkflowRequest, opts ...grpc.CallOption) (*executionv1.GetExecutionWorkflowResponse, error) {
	if f.getExecutionWorkflow == nil {
		return nil, nil
	}
	return f.getExecutionWorkflow(ctx, req, opts...)
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

func TestClientExecuteResponse_RetriesAcceptAfterRedundantStart(t *testing.T) {
	var accepts atomic.Int32
	client := Client{
		client: fakeExecutionUpdatesClient{
			getExecutionWorkflow: func(context.Context, *executionv1.GetExecutionWorkflowRequest, ...grpc.CallOption) (*executionv1.GetExecutionWorkflowResponse, error) {
				return &executionv1.GetExecutionWorkflowResponse{
					Workflow: &testworkflowv1.TestWorkflow{Json: []byte(`{}`)},
				}, nil
			},
			acceptExecution: func(_ context.Context, req *executionv1.AcceptExecutionRequest, _ ...grpc.CallOption) (*executionv1.AcceptExecutionResponse, error) {
				accepts.Add(1)
				require.Equal(t, "exec-1", req.GetExecutionId())
				require.Equal(t, "ns-1", req.GetNamespace())
				return &executionv1.AcceptExecutionResponse{}, nil
			},
		},
		logger: zap.NewNop().Sugar(),
		runner: acceptingRunner{result: &executionworkertypes.ExecuteResult{
			Redundant: true,
			Namespace: "ns-1",
		}},
		callTimeout: time.Second,
	}

	client.executeResponse(context.Background(), &executionv1.GetExecutionUpdatesResponse{
		Start: []*executionv1.ExecutionStart{{
			ExecutionId:   ptr("exec-1"),
			EnvironmentId: ptr("env-1"),
		}},
	})

	require.EqualValues(t, 1, accepts.Load())
}

type acceptingRunner struct {
	result *executionworkertypes.ExecuteResult
	err    error
}

func (r acceptingRunner) Execute(executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	return r.result, r.err
}

func (acceptingRunner) Pause(string) error  { return nil }
func (acceptingRunner) Resume(string) error { return nil }
func (acceptingRunner) Abort(string, string, string) error {
	return nil
}
func (acceptingRunner) Cancel(string, string, string) error {
	return nil
}
