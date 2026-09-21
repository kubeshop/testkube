package orchestration

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedTimeoutable struct {
	deadline time.Time
	finished atomic.Bool
}

func (f *fixedTimeoutable) TimeLeft(now time.Time) *time.Duration {
	d := f.deadline.Sub(now)
	return &d
}

func (f *fixedTimeoutable) IsFinished() bool { return f.finished.Load() }

// within fails the test if fn has not returned inside the limit. A stop that
// never returns would otherwise hang the whole package.
func within(t *testing.T, limit time.Duration, name string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { defer close(done); fn() }()
	select {
	case <-done:
	case <-time.After(limit):
		t.Fatalf("%s did not return within %s, which is a deadlock", name, limit)
	}
}

// TestWatchTimeout_StopCannotDeadlock covers the ways the stop function could
// hang. It blocks rather than returns, so every one of these must finish.
func TestWatchTimeout_StopCannotDeadlock(t *testing.T) {
	t.Run("nothing ever times out", func(t *testing.T) {
		stop := WatchTimeout(func() { t.Fatal("the handler must not run") },
			&fixedTimeoutable{deadline: time.Now().Add(time.Hour)})
		within(t, 2*time.Second, "stop", stop)
	})

	t.Run("no objects to watch", func(t *testing.T) {
		stop := WatchTimeout[*fixedTimeoutable](func() { t.Fatal("the handler must not run") })
		within(t, 2*time.Second, "stop", stop)
	})

	t.Run("the object finished before the deadline", func(t *testing.T) {
		obj := &fixedTimeoutable{deadline: time.Now().Add(300 * time.Millisecond)}
		obj.finished.Store(true)
		stop := WatchTimeout(func() { t.Fatal("the handler must not run") }, obj)
		within(t, 2*time.Second, "stop", stop)
	})

	t.Run("stop is called twice", func(t *testing.T) {
		stop := WatchTimeout(func() {}, &fixedTimeoutable{deadline: time.Now().Add(10 * time.Millisecond)})
		time.Sleep(40 * time.Millisecond)
		within(t, 2*time.Second, "first stop", stop)
		within(t, 2*time.Second, "second stop", stop)
	})

	t.Run("many objects share one handler", func(t *testing.T) {
		var calls atomic.Int32
		objs := make([]*fixedTimeoutable, 20)
		for i := range objs {
			objs[i] = &fixedTimeoutable{deadline: time.Now().Add(10 * time.Millisecond)}
		}
		stop := WatchTimeout(func() { calls.Add(1) }, objs...)
		time.Sleep(60 * time.Millisecond)
		within(t, 2*time.Second, "stop", stop)
		assert.Equal(t, int32(1), calls.Load(), "the handler fires once for the group")
	})

	t.Run("stop races the deadline", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			stop := WatchTimeout(func() { time.Sleep(time.Millisecond) },
				&fixedTimeoutable{deadline: time.Now().Add(time.Millisecond)})
			within(t, 2*time.Second, "stop", stop)
		}
	})

	t.Run("a handler that takes a while still lets stop return", func(t *testing.T) {
		var done atomic.Bool
		stop := WatchTimeout(func() {
			time.Sleep(200 * time.Millisecond)
			done.Store(true)
		}, &fixedTimeoutable{deadline: time.Now().Add(10 * time.Millisecond)})
		time.Sleep(40 * time.Millisecond)
		within(t, 3*time.Second, "stop", stop)
		require.True(t, done.Load(), "stop waited for the handler, which is the point")
	})
}
