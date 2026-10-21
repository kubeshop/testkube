package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var errBusy = errors.New("busy")

func TestPolicy_Do(t *testing.T) {
	permanent := errors.New("permanent")
	tests := []struct {
		name         string
		policy       Policy
		failures     []error
		wantAttempts int
		wantErr      error
	}{
		{
			name:         "a call that works at once runs one time",
			policy:       Policy{Attempts: 3},
			wantAttempts: 1,
		},
		{
			name:         "a call that fails two times runs again until it works",
			policy:       Policy{Attempts: 3, Delay: time.Millisecond},
			failures:     []error{errBusy, errBusy},
			wantAttempts: 3,
		},
		{
			name:         "the last error returns when the attempts run out",
			policy:       Policy{Attempts: 2, Delay: time.Millisecond},
			failures:     []error{errBusy, permanent, errBusy},
			wantAttempts: 2,
			wantErr:      permanent,
		},
		{
			name:         "an error that RetryIf rejects returns at once",
			policy:       Policy{Attempts: 5, Delay: time.Millisecond, RetryIf: func(err error) bool { return errors.Is(err, errBusy) }},
			failures:     []error{permanent},
			wantAttempts: 1,
			wantErr:      permanent,
		},
		{
			name:         "a policy without attempts calls one time",
			policy:       Policy{},
			failures:     []error{errBusy},
			wantAttempts: 1,
			wantErr:      errBusy,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			err := tt.policy.Do(context.Background(), func(attempt int) error {
				assert.Equal(t, calls, attempt)
				calls++
				if attempt < len(tt.failures) {
					return tt.failures[attempt]
				}
				return nil
			})

			assert.Equal(t, tt.wantAttempts, calls)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestPolicy_Do_Context(t *testing.T) {
	tests := []struct {
		name         string
		cancelBefore bool
		wantAttempts int
		wantBusy     bool
	}{
		{name: "a context that ended before the call makes no attempt", cancelBefore: true, wantAttempts: 0},
		{name: "a context that ends during a wait keeps the last error", wantAttempts: 1, wantBusy: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			if tt.cancelBefore {
				cancel()
			}
			calls := 0
			err := Policy{Attempts: 3, Delay: time.Hour}.Do(ctx, func(int) error {
				calls++
				cancel()
				return errBusy
			})

			assert.Equal(t, tt.wantAttempts, calls)
			assert.ErrorIs(t, err, context.Canceled)
			assert.Equal(t, tt.wantBusy, errors.Is(err, errBusy))
		})
	}
}

func TestPolicy_Wait(t *testing.T) {
	tests := []struct {
		name   string
		policy Policy
		want   []time.Duration
	}{
		{
			name:   "a linear wait grows by the delay",
			policy: Policy{Delay: time.Second},
			want:   []time.Duration{time.Second, 2 * time.Second, 3 * time.Second},
		},
		{
			name:   "an exponential wait doubles up to the maximum",
			policy: Policy{Delay: time.Second, Growth: Exponential, MaxDelay: 3 * time.Second},
			want:   []time.Duration{time.Second, 2 * time.Second, 3 * time.Second},
		},
		{
			name:   "a fixed wait stays at the delay",
			policy: Policy{Delay: time.Second, Growth: Fixed},
			want:   []time.Duration{time.Second, time.Second, time.Second},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make([]time.Duration, 0, len(tt.want))
			for n := 1; n <= len(tt.want); n++ {
				got = append(got, tt.policy.wait(n))
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPolicy_Do_OnRetry(t *testing.T) {
	var waits []time.Duration
	policy := Policy{Attempts: 3, Delay: time.Millisecond, OnRetry: func(_ int, _ error, wait time.Duration) {
		waits = append(waits, wait)
	}}

	_ = policy.Do(context.Background(), func(int) error { return errBusy })

	assert.Equal(t, []time.Duration{time.Millisecond, 2 * time.Millisecond}, waits, "OnRetry runs before each wait and not after the last attempt")
}
