package registry

import (
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	kubeReadRetryCount = 3
	kubeReadRetryDelay = 200 * time.Millisecond
)

// kubeReadRetry runs fn against the k8s API and retries transient failures
// (network hiccups, api-server unavailable). Terminal errors that will not
// resolve by retrying — NotFound, Forbidden, Unauthorized — short-circuit.
func kubeReadRetry(fn func() error) error {
	var err error
	for i := 0; i < kubeReadRetryCount; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if k8serrors.IsNotFound(err) || k8serrors.IsForbidden(err) || k8serrors.IsUnauthorized(err) {
			return err
		}
		time.Sleep(time.Duration(i) * kubeReadRetryDelay)
	}
	return err
}
