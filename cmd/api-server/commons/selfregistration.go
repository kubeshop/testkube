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

const (
	selfRegistrationBackoffFirst  = 500 * time.Millisecond
	selfRegistrationBackoffSecond = 1 * time.Second
	selfRegistrationBackoffThird  = 2 * time.Second
	selfRegistrationBackoffFourth = 4 * time.Second

	// selfRegistrationWriteTimeout bounds one Create or Update call, so a hung API server
	// connection counts as one failed attempt instead of blocking startup.
	selfRegistrationWriteTimeout = 10 * time.Second
)

// errSelfRegistrationSecretNotManaged is returned when the secret already exists but Testkube may not overwrite it.
var errSelfRegistrationSecretNotManaged = errors.New("existing secret is not managed by Testkube")

// DefaultSelfRegistrationSecretBackoff returns the wait before each retry. 4 retries over ~7.5s ride out
// transient API server errors (e.g. "connection reset by peer") without delaying startup much.
// Each call returns a fresh slice, so callers cannot alter the defaults.
func DefaultSelfRegistrationSecretBackoff() []time.Duration {
	return []time.Duration{
		selfRegistrationBackoffFirst,
		selfRegistrationBackoffSecond,
		selfRegistrationBackoffThird,
		selfRegistrationBackoffFourth,
	}
}

// SelfRegistrationOptions tunes the retry behaviour of PersistSelfRegistrationSecret.
type SelfRegistrationOptions struct {
	// Backoff is the wait before each retry; its length sets the number of retries.
	Backoff []time.Duration
	// WriteTimeout bounds each Create or Update attempt. Zero means the default (10s).
	WriteTimeout time.Duration
}

// PersistSelfRegistrationSecret stores the runner credentials so the next start reads them
// (via envFrom) instead of registering again. It creates the secret, updates it when it
// already exists, and retries transient failures with a bounded backoff. Each attempt has its
// own timeout; a timed-out attempt is retried while ctx is still alive.
func PersistSelfRegistrationSecret(
	ctx context.Context,
	log SelfRegistrationLogger,
	sm SelfRegistrationSecretWriter,
	namespace, name string,
	data map[string]string,
	opts SelfRegistrationOptions,
) error {
	writeTimeout := opts.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = selfRegistrationWriteTimeout
	}
	backoff := opts.Backoff
	var lastErr error
	attempts := len(backoff) + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, writeTimeout)
		lastErr = writeSelfRegistrationSecret(attemptCtx, sm, namespace, name, data)
		cancel()
		if lastErr == nil {
			if attempt > 1 {
				log.Infow("self-register runner secret saved after retry", "secret", name, "attempt", attempt)
			}
			return nil
		}
		if ctx.Err() != nil {
			return fmt.Errorf("saving secret %q: %w (last error: %v)", name, ctx.Err(), lastErr)
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
	// Without ListAll the manager only sees secrets labelled createdBy=testkube, so a foreign secret
	// of the same name surfaces as ErrNotFound, not ErrNotControlled.
	if errors.Is(err, secretmanager.ErrNotFound) || errors.Is(err, secretmanager.ErrNotControlled) {
		return fmt.Errorf("%w: secret %q exists but lacks label createdBy=testkube; delete it or add the label",
			errSelfRegistrationSecretNotManaged, name)
	}
	return err
}

// isRetryableSecretError reports false for errors that a retry cannot fix (configuration or ownership).
// context.DeadlineExceeded stays retryable: it is a per-attempt timeout, and the caller checks the parent ctx.
func isRetryableSecretError(err error) bool {
	switch {
	case errors.Is(err, secretmanager.ErrCreateDisabled),
		errors.Is(err, secretmanager.ErrModifyDisabled),
		errors.Is(err, secretmanager.ErrManagementDisabled),
		errors.Is(err, secretmanager.ErrNotControlled),
		errors.Is(err, errSelfRegistrationSecretNotManaged),
		k8serrors.IsForbidden(err),
		k8serrors.IsUnauthorized(err),
		k8serrors.IsInvalid(err),
		errors.Is(err, context.Canceled):
		return false
	}
	return true
}
