package controlplaneclient

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func withShortCallBackoff(t *testing.T) {
	t.Helper()
	old := callRetryBaseDelay
	callRetryBaseDelay = time.Millisecond
	t.Cleanup(func() { callRetryBaseDelay = old })
}

// Reproduces the log Sheng reported at pkg/triggers/watcher.go:314 -
// gRPC "no healthy upstream" during a control-plane restart. Without the
// retry, the very next tick (5s later) has to eat the error; with the retry
// we absorb the blip inside the current call and the tick sees a valid list.
func TestCallWithRetry_RetriesOnUnavailableThenSucceeds(t *testing.T) {
	withShortCallBackoff(t)

	var calls int32
	fn := func(_ context.Context, _ struct{}, _ ...grpc.CallOption) (string, error) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return "", status.Error(codes.Unavailable, "no healthy upstream")
		}
		return "OK", nil
	}

	res, err := callWithRetry(context.Background(), metadata.MD{}, fn, struct{}{})

	assert.NoError(t, err)
	assert.Equal(t, "OK", res)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

// DeadlineExceeded is transient in the same way Unavailable is: the caller
// eventually reconnects. Retry so the caller does not see a spurious error.
func TestCallWithRetry_RetriesOnDeadlineExceeded(t *testing.T) {
	withShortCallBackoff(t)

	var calls int32
	fn := func(_ context.Context, _ struct{}, _ ...grpc.CallOption) (string, error) {
		n := atomic.AddInt32(&calls, 1)
		if n < 2 {
			return "", status.Error(codes.DeadlineExceeded, "deadline exceeded")
		}
		return "OK", nil
	}

	res, err := callWithRetry(context.Background(), metadata.MD{}, fn, struct{}{})

	assert.NoError(t, err)
	assert.Equal(t, "OK", res)
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}

// Non-transient codes should exit immediately: retrying a PermissionDenied
// wastes time and hides the real problem from the caller.
func TestCallWithRetry_ShortCircuitsOnNonTransient(t *testing.T) {
	withShortCallBackoff(t)

	var calls int32
	fn := func(_ context.Context, _ struct{}, _ ...grpc.CallOption) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "", status.Error(codes.PermissionDenied, "no perms")
	}

	_, err := callWithRetry(context.Background(), metadata.MD{}, fn, struct{}{})

	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

// If the outage exceeds our budget the caller must see the error so the
// tick logic can decide what to do next (log at warning, re-poll in 5s).
// The propagated error must keep the gRPC code so upstream callers can
// still classify it.
func TestCallWithRetry_GivesUpAfterBudgetAndKeepsCode(t *testing.T) {
	withShortCallBackoff(t)

	var calls int32
	fn := func(_ context.Context, _ struct{}, _ ...grpc.CallOption) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "", status.Error(codes.Unavailable, "no healthy upstream")
	}

	_, err := callWithRetry(context.Background(), metadata.MD{}, fn, struct{}{})

	assert.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Equal(t, int32(callRetryCount), atomic.LoadInt32(&calls))
}

// A canceled context during backoff must abort immediately, not wait out
// the retry delay. This keeps the poll responsive to shutdown.
func TestCallWithRetry_RespectsContextCancellationDuringBackoff(t *testing.T) {
	old := callRetryBaseDelay
	callRetryBaseDelay = 500 * time.Millisecond
	t.Cleanup(func() { callRetryBaseDelay = old })

	var calls int32
	fn := func(_ context.Context, _ struct{}, _ ...grpc.CallOption) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "", status.Error(codes.Unavailable, "no healthy upstream")
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := callWithRetry(ctx, metadata.MD{}, fn, struct{}{})
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed, 400*time.Millisecond, "must not wait out the full backoff after cancel")
	assert.LessOrEqual(t, atomic.LoadInt32(&calls), int32(2))
}
