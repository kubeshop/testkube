package registry

import (
	"context"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	kubeReadRetryCount = 3
	kubeReadRetryDelay = 200 * time.Millisecond
)

// kubeReadRetry runs fn against the k8s API and retries transient failures
// (network hiccups, api-server unavailable). Terminal errors that will not
// resolve by retrying (NotFound, Forbidden, Unauthorized) short-circuit.
//
// The backoff only runs between attempts, never after the last one, and it
// aborts as soon as ctx is cancelled so callers that fan out over many
// namespaces do not eat several seconds of dead wait on shutdown.
func kubeReadRetry(ctx context.Context, fn func() error) error {
	var err error
	for i := 0; i < kubeReadRetryCount; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if k8serrors.IsNotFound(err) || k8serrors.IsForbidden(err) || k8serrors.IsUnauthorized(err) {
			return err
		}
		if i == kubeReadRetryCount-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(i+1) * kubeReadRetryDelay):
		}
	}
	return err
}
