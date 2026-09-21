package orchestration

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWatchTimeout_StopWaitsForTheHandler pins the contract the step runner
// depends on. The handler is where a timed out step gets its status, so a stop
// that returns while the handler still runs lets the next step read that step
// before it has one.
//
// The handler parks on a channel instead of a sleep, so the window it must not
// return in stays open until the test closes it. A correct stop therefore cannot
// return during that window on any machine, and the test cannot fail by accident.
// The test then checks the order the two finished in, so a stop that skips the
// wait has to beat both checks to pass.
func TestWatchTimeout_StopWaitsForTheHandler(t *testing.T) {
	handlerRuns := make(chan struct{})
	releaseHandler := make(chan struct{})
	var order, handlerRank, stopRank atomic.Int32

	stop := WatchTimeout(func() {
		close(handlerRuns)
		<-releaseHandler
		handlerRank.Store(order.Add(1))
	}, expired())

	awaitHandler(t, handlerRuns)

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		stop()
		stopRank.Store(order.Add(1))
	}()

	require.Never(t, isClosed(stopped), 100*time.Millisecond, 5*time.Millisecond,
		"stop returned while the handler was still running")

	close(releaseHandler)

	require.Eventually(t, isClosed(stopped), 5*time.Second, 10*time.Millisecond,
		"stop did not return after the handler finished")

	assert.Equal(t, int32(1), handlerRank.Load(), "the handler finished first")
	assert.Equal(t, int32(2), stopRank.Load(), "stop returned after the handler")
}

type fixedTimeoutable struct {
	deadline time.Time
	finished atomic.Bool
}

// expired builds an object whose deadline already passed, so the watcher calls the
// handler on its first look and the test never waits for a timer.
func expired() *fixedTimeoutable {
	return &fixedTimeoutable{deadline: time.Now().Add(-time.Millisecond)}
}

// isClosed builds a condition that reports whether the channel is closed. It does
// not block, so Eventually and Never can poll it.
func isClosed(ch <-chan struct{}) func() bool {
	return func() bool {
		select {
		case <-ch:
			return true
		default:
			return false
		}
	}
}

// awaitHandler waits until the handler signals that it started. A stop that runs
// before the watcher takes its first look cancels the context and the handler never
// fires, so a test that asserts on the handler must wait for this.
func awaitHandler(t *testing.T, started <-chan struct{}) {
	t.Helper()
	require.Eventually(t, isClosed(started), 5*time.Second, time.Millisecond,
		"the handler never ran, so the assertion that follows proves nothing")
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
	require.Eventually(t, isClosed(done), limit, 5*time.Millisecond,
		"%s did not return within %s, which is a deadlock", name, limit)
}

// TestWatchTimeout_StopCannotDeadlock covers the ways the stop function could
// hang. It blocks rather than returns, so every one of these must finish.
func TestWatchTimeout_StopCannotDeadlock(t *testing.T) {
	tests := []struct {
		name        string
		objects     func() []*fixedTimeoutable
		handlerRuns bool
	}{
		{
			name: "nothing ever times out",
			objects: func() []*fixedTimeoutable {
				return []*fixedTimeoutable{{deadline: time.Now().Add(time.Hour)}}
			},
		},
		{
			name:    "no objects to watch",
			objects: func() []*fixedTimeoutable { return nil },
		},
		{
			name: "the object finished before the deadline",
			objects: func() []*fixedTimeoutable {
				obj := &fixedTimeoutable{deadline: time.Now().Add(300 * time.Millisecond)}
				obj.finished.Store(true)
				return []*fixedTimeoutable{obj}
			},
		},
		{
			name:        "the deadline already passed",
			objects:     func() []*fixedTimeoutable { return []*fixedTimeoutable{expired()} },
			handlerRuns: true,
		},
		{
			name: "many objects share one handler",
			objects: func() []*fixedTimeoutable {
				objs := make([]*fixedTimeoutable, 20)
				for i := range objs {
					objs[i] = expired()
				}
				return objs
			},
			handlerRuns: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			ran := make(chan struct{})
			stop := WatchTimeout(func() {
				if calls.Add(1) == 1 {
					close(ran)
				}
			}, tt.objects()...)

			if tt.handlerRuns {
				awaitHandler(t, ran)
			}
			within(t, 2*time.Second, "stop", stop)
			within(t, 2*time.Second, "the second stop", stop)

			// stop joins every watcher, so the count is final once it returns.
			if tt.handlerRuns {
				assert.Equal(t, int32(1), calls.Load(), "the handler fires once for the group")
			} else {
				assert.Zero(t, calls.Load(), "the handler must not run")
			}
		})
	}

}

// TestWatchTimeout_StopLandsOnTheDeadline stops the watcher as the deadline passes.
// Every case in TestWatchTimeout_StopCannotDeadlock stops it from a state the test
// already knows, so none of them reach the interleaving this one aims at: a watcher
// goroutine between its context check and its call to the handler. The handler holds
// the wait group open long enough that a stop which joins it at the wrong moment hangs
// here rather than passes. One attempt rarely lands on the window, so this samples it.
func TestWatchTimeout_StopLandsOnTheDeadline(t *testing.T) {
	for i := 0; i < 50; i++ {
		stop := WatchTimeout(func() { time.Sleep(time.Millisecond) },
			&fixedTimeoutable{deadline: time.Now().Add(time.Millisecond)})
		within(t, 2*time.Second, "stop", stop)
	}
}
