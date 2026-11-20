package commons

import (
	"context"
	"errors"
	"fmt"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/secretmanager"
)

// SelfRegistrationSecretWriter is the subset of secretmanager.SecretManager needed to persist
// the self-registration credentials.
type SelfRegistrationSecretWriter interface {
	Create(ctx context.Context, namespace, name string, data map[string]string, opts secretmanager.CreateOptions) (testkube.Secret, error)
	Update(ctx context.Context, namespace, name string, data map[string]string, opts secretmanager.UpdateOptions) (testkube.Secret, error)
}

type SelfRegistrationLogger interface {
	Warnw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
}

// SelfRegistrationSecretBackoff is the wait before each retry. 4 retries over ~7.5s rides out
// transient API server errors (e.g. "connection reset by peer") without delaying startup much.
var SelfRegistrationSecretBackoff = []time.Duration{
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
}

// PersistSelfRegistrationSecret stores the runner credentials so the next start reads them
// (via envFrom) instead of registering again. It creates the secret, updates it when it
// already exists, and retries transient failures with a bounded backoff.
func PersistSelfRegistrationSecret(
	ctx context.Context,
	log SelfRegistrationLogger,
	sm SelfRegistrationSecretWriter,
	namespace, name string,
	data map[string]string,
	backoff []time.Duration,
) error {
	var lastErr error
	attempts := len(backoff) + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		lastErr = writeSelfRegistrationSecret(ctx, sm, namespace, name, data)
		if lastErr == nil {
			if attempt > 1 {
				log.Infow("self-register runner secret saved after retry", "secret", name, "attempt", attempt)
			}
			return nil
		}
		if !isRetryableSecretError(lastErr) || attempt == attempts {
			break
		}
		wait := backoff[attempt-1]
		log.Warnw("error saving self-register runner secret, retrying",
			"secret", name, "attempt", attempt, "max_attempts", attempts, "retry_in", wait.String(), "error", lastErr.Error())
		select {
		case <-ctx.Done():
			return fmt.Errorf("saving secret %q: %w (last error: %v)", name, ctx.Err(), lastErr)
		case <-time.After(wait):
		}
	}
	return fmt.Errorf("saving secret %q: %w", name, lastErr)
}

func writeSelfRegistrationSecret(ctx context.Context, sm SelfRegistrationSecretWriter, namespace, name string, data map[string]string) error {
	_, err := sm.Create(ctx, namespace, name, data, secretmanager.CreateOptions{})
	if err == nil || !k8serrors.IsAlreadyExists(err) {
		return err
	}
	// Left over from an earlier attempt or a previous registration: overwrite with the new credentials.
	_, err = sm.Update(ctx, namespace, name, data, secretmanager.UpdateOptions{})
	return err
}

// isRetryableSecretError reports false for errors that a retry cannot fix (configuration or ownership).
func isRetryableSecretError(err error) bool {
	switch {
	case errors.Is(err, secretmanager.ErrCreateDisabled),
		errors.Is(err, secretmanager.ErrModifyDisabled),
		errors.Is(err, secretmanager.ErrManagementDisabled),
		errors.Is(err, secretmanager.ErrNotControlled),
		k8serrors.IsForbidden(err),
		k8serrors.IsUnauthorized(err),
		k8serrors.IsInvalid(err),
		errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return false
	}
	return true
}
