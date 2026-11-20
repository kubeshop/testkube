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
	selfRegistrationWriteTimeout = 5 * time.Second
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
	// WriteTimeout bounds each Create or Update attempt. Zero means the default (5s).
	// With the default backoff the worst case is about 32.5s, well inside the runner startup probe.
	WriteTimeout time.Duration
	// SecretPrefix is the prefix the secret manager adds to name (SECRET_CREATION_PREFIX). It is only
	// used to name the real Kubernetes object in logs and errors.
	SecretPrefix string
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
	return persistSelfRegistrationSecret(ctx, log, sm, namespace, name, data, opts, false)
}

// persistSelfRegistrationSecret implements PersistSelfRegistrationSecret. With alwaysUpdate, a
// successful Create is followed by an Update, so both verbs the post-register write may need are exercised.
func persistSelfRegistrationSecret(
	ctx context.Context,
	log SelfRegistrationLogger,
	sm SelfRegistrationSecretWriter,
	namespace, name string,
	data map[string]string,
	opts SelfRegistrationOptions,
	alwaysUpdate bool,
) error {
	writeTimeout := opts.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = selfRegistrationWriteTimeout
	}
	backoff := opts.Backoff
	fullName := opts.SecretPrefix + name
	var lastErr error
	attempts := len(backoff) + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, writeTimeout)
		lastErr = writeSelfRegistrationSecret(attemptCtx, sm, namespace, name, fullName, data, alwaysUpdate)
		cancel()
		if lastErr == nil {
			if attempt > 1 {
				log.Infow("self-register runner secret saved after retry", "secret", fullName, "attempt", attempt)
			}
			return nil
		}
		if ctx.Err() != nil {
			return fmt.Errorf("saving secret %q: %w (last error: %v)", fullName, ctx.Err(), lastErr)
		}
		if !isRetryableSecretError(lastErr) || attempt == attempts {
			break
		}
		wait := backoff[attempt-1]
		log.Warnw("error saving self-register runner secret, retrying",
			"secret", fullName, "attempt", attempt, "max_attempts", attempts, "retry_in", wait.String(), "error", lastErr.Error())
		select {
		case <-ctx.Done():
			return fmt.Errorf("saving secret %q: %w (last error: %v)", fullName, ctx.Err(), lastErr)
		case <-time.After(wait):
		}
	}
	return fmt.Errorf("saving secret %q: %w", fullName, lastErr)
}

// CheckSelfRegistrationSecretWritable verifies, before the runner registers, that the
// self-registration secret can be written. It creates the secret empty or, when it already exists,
// updates it without touching its data, using the same retries and error classification as
// PersistSelfRegistrationSecret. Configuration, RBAC and ownership problems then surface while the
// Control Plane has no runner yet, so fixing them needs no cleanup in the Control Plane UI.
// It always ends with an Update (also after a successful Create): once the secret exists, the
// post-register write is Create (AlreadyExists) then Update, so a Role without get/update must fail here.
// An empty secret loaded through envFrom leaves TESTKUBE_PRO_API_KEY empty, so the next start
// still registers and PersistSelfRegistrationSecret fills the secret in.
func CheckSelfRegistrationSecretWritable(
	ctx context.Context,
	log SelfRegistrationLogger,
	sm SelfRegistrationSecretWriter,
	namespace, name string,
	opts SelfRegistrationOptions,
) error {
	return persistSelfRegistrationSecret(ctx, log, sm, namespace, name, nil, opts, true)
}

func writeSelfRegistrationSecret(ctx context.Context, sm SelfRegistrationSecretWriter, namespace, name, fullName string, data map[string]string, alwaysUpdate bool) error {
	_, err := sm.Create(ctx, namespace, name, data, secretmanager.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	if err == nil && !alwaysUpdate {
		return nil
	}
	// Left over from an earlier attempt or a previous registration: overwrite with the new credentials.
	// With nil data (the pre-flight check) the secret manager keeps the existing data.
	_, err = sm.Update(ctx, namespace, name, data, secretmanager.UpdateOptions{})
	// Without ListAll the manager only sees secrets labelled createdBy=testkube, so a foreign secret
	// of the same name surfaces as ErrNotFound, not ErrNotControlled. ErrNotFound can also mean the secret
	// was deleted between Create and Update; that race is rare enough to report rather than retry.
	if errors.Is(err, secretmanager.ErrNotFound) || errors.Is(err, secretmanager.ErrNotControlled) {
		return fmt.Errorf("%w: secret %q exists but is not visible to Testkube (missing label createdBy=testkube?); "+
			"delete it or add the label", errSelfRegistrationSecretNotManaged, fullName)
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
