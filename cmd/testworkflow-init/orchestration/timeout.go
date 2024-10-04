package orchestration

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Timeoutable interface {
	TimeLeft(ts time.Time) *time.Duration
	IsFinished() bool
}

func GetTimedOut[T Timeoutable](objects ...T) (result []T) {
	now := time.Now()
	for _, t := range objects {
		// Check if that is still timeoutable
		left := t.TimeLeft(now)
		if left != nil && !t.IsFinished() && *left <= 0 {
			result = append(result, t)
		}
	}
	return result
}

func WatchTimeout[T Timeoutable](handler func(), objects ...T) func() {
	ctx, ctxCancel := context.WithCancel(context.Background())
	var fired atomic.Bool
	run := func() {
		swapped := fired.CompareAndSwap(false, true)
		if swapped {
			handler()
		}
	}

	// Set up timers for all timeoutable objects
	var wg sync.WaitGroup
	wg.Add(len(objects))
	for i := range objects {
		go func(t T) {
			defer wg.Done()
			for {
				// Check if that is still timeoutable
				left := t.TimeLeft(time.Now())
				if left == nil || t.IsFinished() || ctx.Err() != nil {
					return
				}

				// Fire the handler if it timed out
				if *left <= 0 {
					run()
					return
				}

				// Wait until time is finished, or we are no longer waiting
				timer := time.NewTimer(*left)
				select {
				case <-ctx.Done():
					if !timer.Stop() {
						<-timer.C
					}
					return
				case <-timer.C:
				}
			}
		}(objects[i])
	}

	// Cancel if all timeouts are done
	go func() {
		wg.Wait()
		ctxCancel()
	}()

	// Allow to cancel externally
	return ctxCancel
}
