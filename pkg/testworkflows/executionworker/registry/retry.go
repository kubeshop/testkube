package registry

import (
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeshop/testkube/pkg/retry"
)

// kubeReadPolicy retries a read of the Kubernetes API after a transient failure. A missing
// object, a forbidden read and an unauthorized read do not change on a retry, so they return at
// once. A cancel stops the wait, so a caller that reads many namespaces stops at once on shutdown.
var kubeReadPolicy = retry.Policy{
	Attempts: 3,
	Delay:    200 * time.Millisecond,
	RetryIf: func(err error) bool {
		return !k8serrors.IsNotFound(err) && !k8serrors.IsForbidden(err) && !k8serrors.IsUnauthorized(err)
	},
}
