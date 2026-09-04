package registry

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestKubeReadRetry_TransientErrorRetriedThenSucceeds(t *testing.T) {
	var calls int
	transient := stderrors.New("kube-apiserver unavailable")
	err := kubeReadRetry(context.Background(), func() error {
		calls++
		if calls < 2 {
			return transient
		}
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestKubeReadRetry_GivesUpAfterBudget(t *testing.T) {
	var calls int
	transient := stderrors.New("kube-apiserver unavailable")
	err := kubeReadRetry(context.Background(), func() error {
		calls++
		return transient
	})
	assert.ErrorIs(t, err, transient)
	assert.Equal(t, kubeReadRetryCount, calls)
}

func TestKubeReadRetry_NotFoundShortCircuits(t *testing.T) {
	var calls int
	nf := k8serrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "x")
	err := kubeReadRetry(context.Background(), func() error {
		calls++
		return nf
	})
	assert.True(t, k8serrors.IsNotFound(err))
	assert.Equal(t, 1, calls)
}

func TestKubeReadRetry_ForbiddenShortCircuits(t *testing.T) {
	var calls int
	f := k8serrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "x", stderrors.New("nope"))
	err := kubeReadRetry(context.Background(), func() error {
		calls++
		return f
	})
	assert.True(t, k8serrors.IsForbidden(err))
	assert.Equal(t, 1, calls)
}

// Exhausting the budget must not sleep after the final attempt.
// With kubeReadRetryCount=3 and kubeReadRetryDelay=200ms the only real waits
// are before attempts 2 and 3 (200ms + 400ms). A stray sleep after attempt 3
// would push wall-clock past ~600ms.
func TestKubeReadRetry_DoesNotSleepAfterLastAttempt(t *testing.T) {
	transient := stderrors.New("kube-apiserver unavailable")
	start := time.Now()
	err := kubeReadRetry(context.Background(), func() error {
		return transient
	})
	elapsed := time.Since(start)
	assert.ErrorIs(t, err, transient)
	// Expected total wait between attempts: (1+2) * 200ms = 600ms. Allow slack
	// but reject the buggy behaviour where a terminal sleep pushes past 800ms.
	assert.Less(t, elapsed, 800*time.Millisecond, "elapsed=%s", elapsed)
}

// A cancelled context must abort the backoff instead of eating the full delay.
func TestKubeReadRetry_ContextCancelledInterruptsBackoff(t *testing.T) {
	transient := stderrors.New("kube-apiserver unavailable")
	ctx, cancel := context.WithCancel(context.Background())
	var calls int
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	err := kubeReadRetry(ctx, func() error {
		calls++
		return transient
	})
	elapsed := time.Since(start)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Less(t, elapsed, 500*time.Millisecond, "elapsed=%s", elapsed)
	assert.LessOrEqual(t, calls, kubeReadRetryCount)
}
