package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
	testworkflowv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

// stubStartsClient serves the calls a start makes and records the outcome.
type stubStartsClient struct {
	executionv1.TestWorkflowExecutionServiceClient

	// workflowErr, when set, is returned by GetExecutionWorkflow.
	workflowErr error
	// workflowJSON overrides the workflow payload.
	workflowJSON []byte
	// fetchDelay is applied inside GetExecutionWorkflow.
	fetchDelay time.Duration

	mu        sync.Mutex
	declined  map[string]string
	accepted  []string
	inFlight  int32
	maxFlight int32
}

func newStubStartsClient() *stubStartsClient {
	return &stubStartsClient{declined: map[string]string{}}
}

func (s *stubStartsClient) GetExecutionWorkflow(_ context.Context, req *executionv1.GetExecutionWorkflowRequest, _ ...grpc.CallOption) (*executionv1.GetExecutionWorkflowResponse, error) {
	now := atomic.AddInt32(&s.inFlight, 1)
	defer atomic.AddInt32(&s.inFlight, -1)
	for {
		peak := atomic.LoadInt32(&s.maxFlight)
		if now <= peak || atomic.CompareAndSwapInt32(&s.maxFlight, peak, now) {
			break
		}
	}

	if s.fetchDelay > 0 {
		time.Sleep(s.fetchDelay)
	}
	if s.workflowErr != nil {
		return nil, s.workflowErr
	}

	payload := s.workflowJSON
	if payload == nil {
		payload, _ = json.Marshal(testworkflowsv1.TestWorkflow{})
	}
	_ = req
	return &executionv1.GetExecutionWorkflowResponse{
		Workflow: &testworkflowv1.TestWorkflow{Json: payload},
	}, nil
}

func (s *stubStartsClient) DeclineExecution(_ context.Context, req *executionv1.DeclineExecutionRequest, _ ...grpc.CallOption) (*executionv1.DeclineExecutionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.declined[req.GetExecutionId()] = req.GetReason()
	return &executionv1.DeclineExecutionResponse{}, nil
}

func (s *stubStartsClient) AcceptExecution(_ context.Context, req *executionv1.AcceptExecutionRequest, _ ...grpc.CallOption) (*executionv1.AcceptExecutionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accepted = append(s.accepted, req.GetExecutionId())
	return &executionv1.AcceptExecutionResponse{}, nil
}

func (s *stubStartsClient) declines() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.declined))
	for k, v := range s.declined {
		out[k] = v
	}
	return out
}

func (s *stubStartsClient) acceptedIds() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.accepted...)
}

// countingRunner records how many executions were actually started.
type countingRunner struct {
	mu      sync.Mutex
	started []string
	err     error
}

func (r *countingRunner) Execute(req executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	r.mu.Lock()
	r.started = append(r.started, req.Execution.Id)
	r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	return &executionworkertypes.ExecuteResult{Namespace: "testkube"}, nil
}

func (r *countingRunner) Pause(string) error          { return nil }
func (r *countingRunner) Resume(string) error         { return nil }
func (r *countingRunner) Abort(_, _, _ string) error  { return nil }
func (r *countingRunner) Cancel(_, _, _ string) error { return nil }
func (r *countingRunner) startedIds() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.started...)
}

func startsResponse(n int) *executionv1.GetExecutionUpdatesResponse {
	starts := make([]*executionv1.ExecutionStart, 0, n)
	for i := 0; i < n; i++ {
		starts = append(starts, &executionv1.ExecutionStart{
			ExecutionId:   proto.String(fmt.Sprintf("exec-%d", i)),
			EnvironmentId: proto.String("env-1"),
			WorkflowName:  proto.String("wf-a"),
		})
	}
	return &executionv1.GetExecutionUpdatesResponse{Start: starts}
}

func newStartsClient(stub *stubStartsClient, r runner, concurrency int) *Client {
	return &Client{
		client:           stub,
		logger:           zap.NewNop().Sugar(),
		runner:           r,
		callTimeout:      5 * time.Second,
		startConcurrency: concurrency,
	}
}

// The workflow fetch used to run sequentially and inline on the poll goroutine,
// so a batch of N starts cost N round trips before the first execution began.
func TestExecuteResponse_FetchesWorkflowsConcurrently(t *testing.T) {
	stub := newStubStartsClient()
	stub.fetchDelay = 20 * time.Millisecond
	runner := &countingRunner{}
	c := newStartsClient(stub, runner, 4)

	start := time.Now()
	c.executeResponse(context.Background(), startsResponse(8))
	elapsed := time.Since(start)

	assert.Len(t, runner.startedIds(), 8, "every start in the batch has to run")
	assert.Greater(t, int(atomic.LoadInt32(&stub.maxFlight)), 1,
		"the fetches must overlap; sequentially this is stuck at 1")
	// 8 starts at 20ms with 4 in flight is ~40ms; sequentially it would be ~160ms.
	assert.Less(t, elapsed, 140*time.Millisecond,
		"a batch must not cost one round trip per start")
}

func TestExecuteResponse_RespectsStartConcurrencyLimit(t *testing.T) {
	stub := newStubStartsClient()
	stub.fetchDelay = 10 * time.Millisecond
	c := newStartsClient(stub, &countingRunner{}, 3)

	c.executeResponse(context.Background(), startsResponse(12))

	assert.LessOrEqual(t, int(atomic.LoadInt32(&stub.maxFlight)), 3,
		"the limit bounds how much work one batch throws at the cluster at once")
}

// A Client built as a struct literal has a zero concurrency. errgroup.SetLimit(0)
// gives a zero-capacity semaphore, so Go would block forever.
func TestExecuteResponse_ClampsAZeroConcurrency(t *testing.T) {
	stub := newStubStartsClient()
	runner := &countingRunner{}
	c := newStartsClient(stub, runner, 0)

	done := make(chan struct{})
	go func() {
		defer close(done)
		c.executeResponse(context.Background(), startsResponse(3))
	}()

	select {
	case <-done:
		assert.Len(t, runner.startedIds(), 3)
	case <-time.After(5 * time.Second):
		t.Fatal("executeResponse blocked on a zero concurrency limit")
	}
}

func TestExecuteResponse_AcceptsStartedExecutions(t *testing.T) {
	stub := newStubStartsClient()
	c := newStartsClient(stub, &countingRunner{}, 4)

	c.executeResponse(context.Background(), startsResponse(3))

	assert.Len(t, stub.acceptedIds(), 3, "the control plane has to be told each execution is live")
	assert.Empty(t, stub.declines())
}

// A workflow that cannot be parsed will not parse on the next poll either, so
// the control plane has to be told. Otherwise the row stays STARTING and is
// re-offered forever, taking a slot in every batch behind it.
func TestExecuteResponse_DeclinesOnUnmarshalFailure(t *testing.T) {
	stub := newStubStartsClient()
	stub.workflowJSON = []byte("not-json")
	runner := &countingRunner{}
	c := newStartsClient(stub, runner, 4)

	c.executeResponse(context.Background(), startsResponse(1))

	assert.Empty(t, runner.startedIds(), "nothing may be started from a workflow we could not read")
	declines := stub.declines()
	require.Contains(t, declines, "exec-0")
	assert.Equal(t, "definition-invalid", declines["exec-0"])
}

// A transient failure is the burst symptom in this very issue. Declining on it
// would permanently fail executions because the control plane was briefly busy;
// the next poll re-sends them instead.
func TestExecuteResponse_DoesNotDeclineOnTransientFetchFailure(t *testing.T) {
	for _, code := range []codes.Code{codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted} {
		t.Run(code.String(), func(t *testing.T) {
			stub := newStubStartsClient()
			stub.workflowErr = status.Error(code, "try again")
			c := newStartsClient(stub, &countingRunner{}, 4)

			c.executeResponse(context.Background(), startsResponse(1))

			assert.Empty(t, stub.declines(), "a transient failure must be left for the next poll")
		})
	}
}

func TestExecuteResponse_DeclinesOnPermanentFetchFailure(t *testing.T) {
	for _, code := range []codes.Code{codes.NotFound, codes.PermissionDenied, codes.InvalidArgument} {
		t.Run(code.String(), func(t *testing.T) {
			stub := newStubStartsClient()
			stub.workflowErr = status.Error(code, "no")
			c := newStartsClient(stub, &countingRunner{}, 4)

			c.executeResponse(context.Background(), startsResponse(1))

			declines := stub.declines()
			require.Contains(t, declines, "exec-0",
				"a permanent failure must be reported, or the execution is re-offered forever")
		})
	}
}

// The poll loop relies on this blocking, or it would poll faster than it can
// process responses.
func TestExecuteResponse_BlocksUntilTheBatchCompletes(t *testing.T) {
	stub := newStubStartsClient()
	stub.fetchDelay = 5 * time.Millisecond
	runner := &countingRunner{}
	c := newStartsClient(stub, runner, 2)

	c.executeResponse(context.Background(), startsResponse(6))

	assert.Len(t, runner.startedIds(), 6, "every start has finished by the time executeResponse returns")
	assert.Len(t, stub.acceptedIds(), 6)
}

// BenchmarkExecuteResponse is a latency curve, not a CPU measurement: the fetch
// delay is injected. It shows N*RTT collapsing to ceil(N/limit)*RTT, and puts a
// number behind the default concurrency.
func BenchmarkExecuteResponse(b *testing.B) {
	for _, batch := range []int{1, 10, 25, 50} {
		for _, limit := range []int{1, 4, 8} {
			b.Run(fmt.Sprintf("batch=%d/limit=%d", batch, limit), func(b *testing.B) {
				response := startsResponse(batch)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					stub := newStubStartsClient()
					stub.fetchDelay = time.Millisecond
					c := newStartsClient(stub, &countingRunner{}, limit)
					c.executeResponse(context.Background(), response)
				}
			})
		}
	}
}
