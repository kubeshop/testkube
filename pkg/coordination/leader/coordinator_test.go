package leader

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type leaseResponse struct {
	leased bool
	err    error
}

type stubLeaseBackend struct {
	responses chan leaseResponse
	calls     atomic.Int32
}

func newStubLeaseBackend(buffer int) *stubLeaseBackend {
	return &stubLeaseBackend{
		responses: make(chan leaseResponse, buffer),
	}
}

func (s *stubLeaseBackend) TryAcquire(ctx context.Context, _, _ string) (bool, error) {
	s.calls.Add(1)
	select {
	case res := <-s.responses:
		return res.leased, res.err
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func (s *stubLeaseBackend) push(res leaseResponse) {
	s.responses <- res
}

func TestCoordinatorStartsTasksWhenLeaseAcquired(t *testing.T) {
	backend := newStubLeaseBackend(1)
	backend.push(leaseResponse{leased: true})

	var started atomic.Int32

	task := Task{
		Name: "starter",
		Start: func(ctx context.Context) error {
			started.Add(1)
			<-ctx.Done()
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	coord := New(backend, "id-1", "cluster-1", nil, WithCheckInterval(10*time.Millisecond))
	coord.Register(task)

	done := make(chan struct{})
	go func() {
		if err := coord.Run(ctx); err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
		close(done)
	}()

	waitFor(func() bool { return started.Load() > 0 }, 200*time.Millisecond, t)

	cancel()
	waitFor(func() bool { return isClosed(done) }, 200*time.Millisecond, t)

	if got := started.Load(); got != 1 {
		t.Fatalf("expected task to start once, got %d", got)
	}
}

func TestCoordinatorStopsTasksWhenLeaseLost(t *testing.T) {
	backend := newStubLeaseBackend(3)
	backend.push(leaseResponse{leased: true})
	backend.push(leaseResponse{leased: true})
	backend.push(leaseResponse{leased: false})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopped := make(chan struct{})
	task := Task{
		Name: "stopper",
		Start: func(tctx context.Context) error {
			<-tctx.Done()
			close(stopped)
			return nil
		},
	}

	coord := New(backend, "id-1", "cluster-1", nil, WithCheckInterval(10*time.Millisecond))
	coord.Register(task)

	go func() {
		_ = coord.Run(ctx)
	}()

	waitFor(func() bool { return isClosed(stopped) }, 500*time.Millisecond, t)
}

func TestCoordinatorRestartsTasksAfterReacquire(t *testing.T) {
	backend := newStubLeaseBackend(5)
	backend.push(leaseResponse{leased: true})
	backend.push(leaseResponse{leased: false})
	backend.push(leaseResponse{leased: true})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	startCount := 0
	stopCount := 0

	task := Task{
		Name: "flappy",
		Start: func(tctx context.Context) error {
			mu.Lock()
			startCount++
			mu.Unlock()

			<-tctx.Done()

			mu.Lock()
			stopCount++
			mu.Unlock()
			return nil
		},
	}

	coord := New(backend, "id-1", "cluster-1", nil, WithCheckInterval(10*time.Millisecond))
	coord.Register(task)

	done := make(chan struct{})
	go func() {
		_ = coord.Run(ctx)
		close(done)
	}()

	waitFor(func() bool {
		mu.Lock()
		defer mu.Unlock()
		return startCount >= 2 && stopCount >= 1
	}, 500*time.Millisecond, t)

	cancel()
	waitFor(func() bool { return isClosed(done) }, 200*time.Millisecond, t)

	mu.Lock()
	defer mu.Unlock()
	if startCount < 2 {
		t.Fatalf("expected task to start at least twice, got %d", startCount)
	}
	if stopCount < 1 {
		t.Fatalf("expected task to stop at least once, got %d", stopCount)
	}
}

func TestCoordinatorTreatsErrorsAsLeaseLoss(t *testing.T) {
	backend := newStubLeaseBackend(3)
	backend.push(leaseResponse{leased: true})
	backend.push(leaseResponse{err: errors.New("boom")})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopped := make(chan struct{})
	task := Task{
		Name: "error",
		Start: func(tctx context.Context) error {
			<-tctx.Done()
			close(stopped)
			return nil
		},
	}

	coord := New(backend, "id-1", "cluster-1", nil, WithCheckInterval(10*time.Millisecond))
	coord.Register(task)

	go func() {
		_ = coord.Run(ctx)
	}()

	waitFor(func() bool { return isClosed(stopped) }, 500*time.Millisecond, t)
}

func waitFor(cond func() bool, timeout time.Duration, t *testing.T) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !cond() {
		t.Fatalf("condition not met within %s", timeout.String())
	}
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
