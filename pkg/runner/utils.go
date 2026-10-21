package runner

import (
	"context"
	"time"

	retrypkg "github.com/kubeshop/testkube/pkg/retry"
)

// retry calls fn up to count times. The wait before attempt n+1 is n times delayBase, no wait
// follows the last attempt, and a context that ends stops the wait.
func retry(ctx context.Context, count int, delayBase time.Duration, fn func(retryCount int) error) error {
	return retrypkg.Policy{Attempts: count, Delay: delayBase}.Do(ctx, fn)
}
