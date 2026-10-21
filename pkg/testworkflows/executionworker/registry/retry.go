package registry

import (
	"context"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeshop/testkube/pkg/retry"
)

// kubeReadPolicy retries a read of the k8s API after a transient failure (network hiccups, api-server
// unavailable). Terminal errors that will not resolve by retrying (NotFound, Forbidden,
// Unauthorized) short-circuit. The wait aborts as soon as ctx is cancelled, so callers that fan out
// over many namespaces do not eat several seconds of dead wait on shutdown.
var kubeReadPolicy = retry.Policy{
	Attempts: 3,
	Delay:    200 * time.Millisecond,
	RetryIf: func(err error) bool {
		return !k8serrors.IsNotFound(err) && !k8serrors.IsForbidden(err) && !k8serrors.IsUnauthorized(err)
	},
}

// kubeReadRetry runs fn against the k8s API under kubeReadPolicy.
func kubeReadRetry(ctx context.Context, fn func() error) error {
	return kubeReadPolicy.Do(ctx, func(int) error { return fn() })
}
