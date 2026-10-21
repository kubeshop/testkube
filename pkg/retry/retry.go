package retry

import (
	"context"
	"errors"
	"time"

	retrygo "github.com/avast/retry-go/v5"
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
// until the context ends. The attempt that fn gets starts at 0.
func (p Policy) Do(ctx context.Context, fn func(attempt int) error) error {
	attempts := max(p.Attempts, 1)
	retryIf := func(err error) bool { return p.RetryIf == nil || p.RetryIf(err) }

	var last error
	attempt := 0
	err := retrygo.New(
		retrygo.Context(ctx),
		retrygo.Attempts(uint(attempts)),
		retrygo.Delay(p.Delay),
		retrygo.DelayType(func(n uint, _ error, _ retrygo.DelayContext) time.Duration { return p.wait(int(n)) }),
		retrygo.RetryIf(retryIf),
		retrygo.LastErrorOnly(true),
		retrygo.OnRetry(func(n uint, err error) {
			if p.OnRetry != nil && int(n)+1 < attempts {
				p.OnRetry(int(n), err, p.wait(int(n)+1))
			}
		}),
	).Do(func() error {
		last = fn(attempt)
		attempt++
		return last
	})

	// retry-go returns only the cause of the context when it ends during a wait.
	if err != nil && last != nil && ctx.Err() != nil && !errors.Is(err, last) {
		return errors.Join(last, err)
	}
	return err
}

// wait returns the wait before attempt n+1, for n from 1.
func (p Policy) wait(n int) time.Duration {
	var d time.Duration
	switch p.Growth {
	case Exponential:
		d = p.Delay << (n - 1)
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
