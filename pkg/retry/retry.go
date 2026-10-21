package retry

import (
	"context"
	"errors"
	"time"
)

// Growth is how the wait changes from one attempt to the next.
type Growth int

const (
	// Linear grows the wait by the delay after each attempt.
	Linear Growth = iota
	// Exponential doubles the wait after each attempt.
	Exponential
	// Fixed keeps the wait at the delay.
	Fixed
)

// Policy is how a bounded call is retried.
type Policy struct {
	// Attempts is the number of calls, the first one included. A value below 1 means one call.
	Attempts int
	// Delay is the wait before the second attempt.
	Delay time.Duration
	// Growth is how the wait changes. The default is Linear.
	Growth Growth
	// MaxDelay limits one wait. Zero means no limit.
	MaxDelay time.Duration
	// RetryIf reports whether an error is worth another attempt. Nil retries every error.
	RetryIf func(error) bool
	// OnRetry runs after a failed attempt that gets another one, before the wait. The attempt
	// starts at 0.
	OnRetry func(attempt int, err error, wait time.Duration)
}

// Do calls fn until it succeeds, until RetryIf rejects its error, until the attempts run out, or
// until the context ends.
func (p Policy) Do(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	attempts := max(p.Attempts, 1)
	var err error
	for attempt := range attempts {
		if err = fn(); err == nil {
			return nil
		}
		if attempt == attempts-1 || (p.RetryIf != nil && !p.RetryIf(err)) {
			return err
		}
		wait := p.wait(attempt + 1)
		if p.OnRetry != nil {
			p.OnRetry(attempt, err, wait)
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(err, context.Cause(ctx))
		case <-timer.C:
		}
	}
	return err
}

// maxShift keeps an exponential wait inside time.Duration.
const maxShift = 62

// wait returns the wait before attempt n+1, for n from 1.
func (p Policy) wait(n int) time.Duration {
	var d time.Duration
	switch p.Growth {
	case Exponential:
		shift := min(n-1, maxShift)
		d = p.Delay << shift
		if d>>shift != p.Delay {
			d = time.Duration(1<<63 - 1)
		}
	case Fixed:
		d = p.Delay
	default:
		d = time.Duration(n) * p.Delay
	}
	if p.MaxDelay > 0 && d > p.MaxDelay {
		d = p.MaxDelay
	}
	return d
}
