package orchestration

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type slowTimeoutable struct {
	deadline time.Time
}

func (s slowTimeoutable) TimeLeft(now time.Time) *time.Duration {
	d := s.deadline.Sub(now)
	return &d
}

func (s slowTimeoutable) IsFinished() bool { return false }

// TestWatchTimeout_StopWaitsForTheHandler pins the contract the step runner
// depends on. The handler is where a timed out step gets its status, so a stop
// that returns while the handler still runs lets the next step read that step
// before it has one.
func TestWatchTimeout_StopWaitsForTheHandler(t *testing.T) {
	var handlerDone atomic.Bool

	stop := WatchTimeout(func() {
		// Stand in for the work the real handler does before the runner moves on.
		time.Sleep(150 * time.Millisecond)
		handlerDone.Store(true)
	}, slowTimeoutable{deadline: time.Now().Add(10 * time.Millisecond)})

	// Let the timer fire so the handler is running when stop is called.
	time.Sleep(50 * time.Millisecond)
	stop()

	assert.True(t, handlerDone.Load(), "stop must not return while the handler is still running")
}
