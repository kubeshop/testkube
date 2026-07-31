package leader

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const waitTimeout = time.Second

type leaseResponse struct {
	leased bool
	err    error
}

// scriptedLeaseBackend replays a fixed sequence of TryAcquire results, repeating the last entry
// once the script is exhausted. An empty script always reports the lease as held by someone else.
type scriptedLeaseBackend struct {
	mu        sync.Mutex
	responses []leaseResponse
	calls     int
	// onCall runs on every TryAcquire, so a test can model time passing inside the call.
	onCall func()
}

func newScriptedLeaseBackend(responses ...leaseResponse) *scriptedLeaseBackend {
	return &scriptedLeaseBackend{responses: responses}
}

func (s *scriptedLeaseBackend) TryAcquire(_ context.Context, _, _ string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	if s.onCall != nil {
		s.onCall()
	}
	if len(s.responses) == 0 {
		return false, nil
	}

	index := s.calls - 1
	if index >= len(s.responses) {
		index = len(s.responses) - 1
	}
	return s.responses[index].leased, s.responses[index].err
}

func (s *scriptedLeaseBackend) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.calls
}

type fakeClock struct {
	mu      sync.Mutex
	current time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.current
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.current = c.current.Add(d)
}

type countingTask struct {
	starts atomic.Int32
	stops  atomic.Int32
}

func (c *countingTask) task() Task {
	return Task{
		Name: "counting",
		Start: func(ctx context.Context) error {
			c.starts.Add(1)
			<-ctx.Done()
			c.stops.Add(1)
			return nil
		},
	}
}

func TestCoordinator_evaluate(t *testing.T) {
	const leaseDuration = time.Minute

	backendErr := errors.New("backend unreachable")

	tests := []struct {
		name        string
		responses   []leaseResponse
		evaluations int
		tick        time.Duration
		// callDuration is how much time passes inside each lease check.
		callDuration time.Duration
		wantStarts   int32
		wantStops    int32
		wantLeader   bool
		wantFailures int
	}{
		{
			name:        "acquires leadership and starts tasks",
			responses:   []leaseResponse{{leased: true}},
			evaluations: 3,
			tick:        5 * time.Second,
			wantStarts:  1,
			wantStops:   0,
			wantLeader:  true,
		},
		{
			name:         "keeps leadership while errors stay within the lease duration",
			responses:    []leaseResponse{{leased: true}, {err: backendErr}, {err: backendErr}, {leased: true}},
			evaluations:  4,
			tick:         5 * time.Second,
			wantStarts:   1,
			wantStops:    0,
			wantLeader:   true,
			wantFailures: 0,
		},
		{
			name:         "releases leadership once errors outlive the lease duration",
			responses:    []leaseResponse{{leased: true}, {err: backendErr}},
			evaluations:  6,
			tick:         20 * time.Second,
			wantStarts:   1,
			wantStops:    1,
			wantLeader:   false,
			wantFailures: 5,
		},
		{
			name:        "releases leadership immediately when another holder is reported",
			responses:   []leaseResponse{{leased: true}, {leased: false}},
			evaluations: 2,
			tick:        5 * time.Second,
			wantStarts:  1,
			wantStops:   1,
			wantLeader:  false,
		},
		{
			// A slow check must not extend the window: the renewal counts from when the check
			// started, not from when the backend answered.
			name:         "measures the tolerance window from the start of the lease check",
			responses:    []leaseResponse{{leased: true}, {err: backendErr}},
			evaluations:  2,
			tick:         0,
			callDuration: leaseDuration / 2,
			wantStarts:   1,
			wantStops:    1,
			wantLeader:   false,
			wantFailures: 1,
		},
		{
			name:         "does not acquire leadership when checks fail from the start",
			responses:    []leaseResponse{{err: backendErr}},
			evaluations:  2,
			tick:         5 * time.Second,
			wantStarts:   0,
			wantStops:    0,
			wantLeader:   false,
			wantFailures: 2,
		},
		{
			name:         "restarts tasks after leadership is regained",
			responses:    []leaseResponse{{leased: true}, {leased: false}, {leased: true}},
			evaluations:  3,
			tick:         5 * time.Second,
			wantStarts:   2,
			wantStops:    1,
			wantLeader:   true,
			wantFailures: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			clock := &fakeClock{current: time.Now()}
			backend := newScriptedLeaseBackend(tt.responses...)
			if tt.callDuration > 0 {
				backend.onCall = func() { clock.advance(tt.callDuration) }
			}
			task := &countingTask{}

			coord := New(backend, "id-1", "cluster-1", nil,
				WithLeaseDuration(leaseDuration),
				withClock(clock.now))
			coord.Register(task.task())

			for i := 0; i < tt.evaluations; i++ {
				clock.advance(tt.tick)
				coord.evaluate(ctx)
			}

			require.Eventually(t, func() bool {
				return task.starts.Load() == tt.wantStarts
			}, waitTimeout, 5*time.Millisecond, "expected %d task starts", tt.wantStarts)

			if got := task.stops.Load(); got != tt.wantStops {
				t.Fatalf("expected %d task stops, got %d", tt.wantStops, got)
			}

			coord.mu.Lock()
			gotLeader, gotFailures := coord.leader, coord.consecutiveFailures
			coord.mu.Unlock()

			if gotLeader != tt.wantLeader {
				t.Fatalf("expected leader %v, got %v", tt.wantLeader, gotLeader)
			}
			if gotFailures != tt.wantFailures {
				t.Fatalf("expected %d consecutive failures, got %d", tt.wantFailures, gotFailures)
			}
		})
	}
}

func TestCoordinator_Run(t *testing.T) {
	backendErr := errors.New("backend unreachable")

	tests := []struct {
		name          string
		responses     []leaseResponse
		disabled      bool
		leaseDuration time.Duration
		// waitCalls holds the run loop until the backend has been consulted that many times.
		waitCalls          int
		minStops           int32
		wantStarts         int32
		expectBackendCalls bool
	}{
		{
			name:               "starts tasks once while the lease is held",
			responses:          []leaseResponse{{leased: true}},
			wantStarts:         1,
			expectBackendCalls: true,
		},
		{
			name:               "stops tasks when the lease is taken by another holder",
			responses:          []leaseResponse{{leased: true}, {leased: true}, {leased: false}},
			minStops:           1,
			wantStarts:         1,
			expectBackendCalls: true,
		},
		{
			name:               "restarts tasks after re-acquiring the lease",
			responses:          []leaseResponse{{leased: true}, {leased: false}, {leased: true}},
			minStops:           1,
			wantStarts:         2,
			expectBackendCalls: true,
		},
		{
			name:               "keeps tasks running while lease checks fail within the lease duration",
			responses:          []leaseResponse{{leased: true}, {err: backendErr}},
			leaseDuration:      time.Minute,
			waitCalls:          5,
			wantStarts:         1,
			expectBackendCalls: true,
		},
		{
			name:               "stops tasks when lease checks keep failing past the lease duration",
			responses:          []leaseResponse{{leased: true}, {err: backendErr}},
			leaseDuration:      50 * time.Millisecond,
			minStops:           1,
			wantStarts:         1,
			expectBackendCalls: true,
		},
		{
			name:               "runs tasks without touching the backend when election is disabled",
			disabled:           true,
			wantStarts:         1,
			expectBackendCalls: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			backend := newScriptedLeaseBackend(tt.responses...)
			task := &countingTask{}

			coord := New(backend, "id-1", "cluster-1", nil,
				WithCheckInterval(10*time.Millisecond),
				WithLeaseDuration(tt.leaseDuration),
				WithLeaderElectionDisabled(tt.disabled))
			coord.Register(task.task())

			done := make(chan struct{})
			go func() {
				if err := coord.Run(ctx); !errors.Is(err, context.Canceled) {
					t.Errorf("expected context.Canceled, got %v", err)
				}
				close(done)
			}()

			require.Eventually(t, func() bool {
				return task.starts.Load() >= tt.wantStarts && task.stops.Load() >= tt.minStops
			}, waitTimeout, 5*time.Millisecond, "tasks to start and stop")

			if tt.waitCalls > 0 {
				require.Eventually(t, func() bool {
					return backend.callCount() >= tt.waitCalls
				}, waitTimeout, 5*time.Millisecond, "expected %d backend calls", tt.waitCalls)
			}

			cancel()
			select {
			case <-done:
			case <-time.After(waitTimeout):
				t.Fatalf("run did not return within %s of cancellation", waitTimeout)
			}

			if got := task.starts.Load(); got != tt.wantStarts {
				t.Fatalf("expected %d task starts, got %d", tt.wantStarts, got)
			}
			if got := backend.callCount(); tt.expectBackendCalls != (got > 0) {
				t.Fatalf("expected backend contacted %v, got %d calls", tt.expectBackendCalls, got)
			}
		})
	}
}
