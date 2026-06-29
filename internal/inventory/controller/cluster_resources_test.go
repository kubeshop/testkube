package controller

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// fakeDiscoverer implements ClusterResourcesDiscoverer with caller-controlled
// responses. listFn lets tests change the return per call.
type fakeDiscoverer struct {
	mu     sync.Mutex
	calls  int32
	listFn func(call int32) ([]testkube.ClusterResource, error)
}

func (f *fakeDiscoverer) List(ctx context.Context) ([]testkube.ClusterResource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := atomic.AddInt32(&f.calls, 1)
	if f.listFn != nil {
		return f.listFn(c)
	}
	return nil, nil
}

// fakePusher implements ClusterResourcesPusher and captures every snapshot it
// was asked to push, plus a configurable per-call error.
type fakePusher struct {
	mu        sync.Mutex
	snapshots [][]testkube.ClusterResource
	pushErr   error
}

func (p *fakePusher) PutClusterResources(_ context.Context, resources []testkube.ClusterResource) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.snapshots = append(p.snapshots, resources)
	return p.pushErr
}

func (p *fakePusher) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.snapshots)
}

func (p *fakePusher) lastSnapshot() []testkube.ClusterResource {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.snapshots) == 0 {
		return nil
	}
	return p.snapshots[len(p.snapshots)-1]
}

func newController(d *fakeDiscoverer, p *fakePusher, interval time.Duration) *ClusterResourcesController {
	return &ClusterResourcesController{
		Discoverer: d,
		Pusher:     p,
		Interval:   interval,
		Log:        zap.NewNop().Sugar(),
	}
}

// waitForCallCount polls until the pusher has recorded at least `want` calls
// or the deadline elapses.
func waitForCallCount(p *fakePusher, want int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p.callCount() >= want {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return p.callCount() >= want
}

func TestPushOnceFiltersWatchableOnly(t *testing.T) {
	d := &fakeDiscoverer{listFn: func(int32) ([]testkube.ClusterResource, error) {
		return []testkube.ClusterResource{
			{Group: "", Version: "v1", Kind: "Pod", CanWatch: true},
			{Group: "", Version: "v1", Kind: "Secret", CanWatch: false},
			{Group: "argoproj.io", Version: "v1alpha1", Kind: "Rollout", CanWatch: true},
		}, nil
	}}
	p := &fakePusher{}
	c := newController(d, p, 0)

	c.pushOnce(context.Background())

	assert.Equal(t, 1, p.callCount())
	got := p.lastSnapshot()
	assert.Len(t, got, 2)
	for _, r := range got {
		assert.True(t, r.CanWatch, "unwatchable resource leaked into push: %v", r)
	}
}

func TestPushOnceSkipsPushWhenDiscoveryFails(t *testing.T) {
	d := &fakeDiscoverer{listFn: func(int32) ([]testkube.ClusterResource, error) {
		return nil, errors.New("discovery boom")
	}}
	p := &fakePusher{}
	c := newController(d, p, 0)

	c.pushOnce(context.Background())

	assert.Equal(t, 0, p.callCount(), "push must not run when discovery returns an error")
}

func TestPushOnceContinuesWhenPushFails(t *testing.T) {
	d := &fakeDiscoverer{listFn: func(int32) ([]testkube.ClusterResource, error) {
		return []testkube.ClusterResource{{Kind: "Pod", CanWatch: true}}, nil
	}}
	p := &fakePusher{pushErr: errors.New("cp unavailable")}
	c := newController(d, p, 0)

	// Calling twice in a row exercises the "next tick retries" path: the first
	// call's push error must NOT be propagated to the caller, and the second
	// call must still run.
	c.pushOnce(context.Background())
	c.pushOnce(context.Background())

	assert.Equal(t, int32(2), atomic.LoadInt32(&d.calls))
	assert.Equal(t, 2, p.callCount())
}

func TestPushOnceWithEmptyResourceList(t *testing.T) {
	// Discovery legitimately returns empty (e.g., RBAC denies everything).
	// We should still push (CP needs to know there are no watchables) - but
	// the snapshot is an empty slice, not nil.
	d := &fakeDiscoverer{listFn: func(int32) ([]testkube.ClusterResource, error) {
		return []testkube.ClusterResource{}, nil
	}}
	p := &fakePusher{}
	c := newController(d, p, 0)

	c.pushOnce(context.Background())

	assert.Equal(t, 1, p.callCount())
	assert.Empty(t, p.lastSnapshot())
}

func TestRunPushesOnStartupThenOnTick(t *testing.T) {
	d := &fakeDiscoverer{listFn: func(int32) ([]testkube.ClusterResource, error) {
		return []testkube.ClusterResource{{Kind: "Pod", CanWatch: true}}, nil
	}}
	p := &fakePusher{}
	c := newController(d, p, 30*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = c.Run(ctx)
		close(done)
	}()

	// Wait until the startup push lands AND at least one tick fires.
	if !waitForCallCount(p, 2, time.Second) {
		cancel()
		<-done
		t.Fatalf("expected at least 2 pushes within deadline, got %d", p.callCount())
	}

	cancel()
	<-done
}

func TestRunReturnsCleanlyOnContextCancel(t *testing.T) {
	d := &fakeDiscoverer{listFn: func(int32) ([]testkube.ClusterResource, error) {
		return nil, nil
	}}
	p := &fakePusher{}
	// Long interval so we exercise the cancellation path, not the tick path.
	c := newController(d, p, 24*time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()

	// Let the startup push happen, then cancel.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatalf("Run did not return within 1s of context cancel")
	}
	// startup push happened; no second tick because of long interval.
	assert.Equal(t, 1, p.callCount())
}

func TestRunPushesOnNotifierAfterDebounce(t *testing.T) {
	d := &fakeDiscoverer{listFn: func(int32) ([]testkube.ClusterResource, error) {
		return []testkube.ClusterResource{{Kind: "Pod", CanWatch: true}}, nil
	}}
	p := &fakePusher{}
	notifier := make(chan struct{}, 4)
	c := newController(d, p, time.Hour) // long tick - the test exercises the notifier path only.
	c.Notifier = notifier
	c.Debounce = 30 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = c.Run(ctx); close(done) }()

	// Wait for the startup push so the count is at a known baseline.
	require.True(t, waitForCallCount(p, 1, time.Second), "startup push did not land")

	// Single notifier event → one push after debounce window.
	notifier <- struct{}{}
	if !waitForCallCount(p, 2, time.Second) {
		cancel()
		<-done
		t.Fatalf("expected notifier-triggered push within 1s, got %d total pushes", p.callCount())
	}

	cancel()
	<-done
	assert.Equal(t, 2, p.callCount(), "expected exactly startup + 1 notifier push")
}

func TestRunCoalescesNotifierBurst(t *testing.T) {
	d := &fakeDiscoverer{listFn: func(int32) ([]testkube.ClusterResource, error) {
		return []testkube.ClusterResource{{Kind: "Pod", CanWatch: true}}, nil
	}}
	p := &fakePusher{}
	notifier := make(chan struct{}, 16)
	c := newController(d, p, time.Hour)
	c.Notifier = notifier
	c.Debounce = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = c.Run(ctx); close(done) }()

	// Wait for startup.
	require.True(t, waitForCallCount(p, 1, time.Second), "startup push did not land")

	// Fire a burst of 10 events spaced inside the debounce window. Each event
	// resets the timer, so only one push should fire after the burst quiets.
	for i := 0; i < 10; i++ {
		notifier <- struct{}{}
		time.Sleep(5 * time.Millisecond)
	}

	// Wait long enough for the debounce timer to actually fire after the burst.
	time.Sleep(150 * time.Millisecond)

	cancel()
	<-done
	assert.Equal(t, 2, p.callCount(), "burst of 10 events must coalesce into 1 push (plus startup)")
}

func TestRunNotifierWithoutChannelDoesNothingExtra(t *testing.T) {
	// Sanity: when Notifier is nil, the existing tick + startup behavior is
	// preserved unchanged.
	d := &fakeDiscoverer{listFn: func(int32) ([]testkube.ClusterResource, error) {
		return nil, nil
	}}
	p := &fakePusher{}
	c := newController(d, p, time.Hour) // no notifier set

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = c.Run(ctx); close(done) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	assert.Equal(t, 1, p.callCount(), "without notifier, only startup push fires within 20ms")
}

func TestDefaultIntervalUsedWhenZero(t *testing.T) {
	// We can't observe a 1h tick, but we can verify the controller runs at all
	// with a zero interval - the interval == 0 branch must initialize without
	// panic and the startup push must still fire.
	d := &fakeDiscoverer{listFn: func(int32) ([]testkube.ClusterResource, error) {
		return nil, nil
	}}
	p := &fakePusher{}
	c := newController(d, p, 0)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = c.Run(ctx)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	assert.Equal(t, 1, p.callCount(), "startup push must fire even with interval=0")
}
