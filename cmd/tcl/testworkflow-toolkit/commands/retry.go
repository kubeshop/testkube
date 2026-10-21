// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package commands

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilnet "k8s.io/apimachinery/pkg/util/net"

	"github.com/kubeshop/testkube/pkg/retry"
)

const cleanupRetryCount = 5

// cleanupRetryDelay is a var so tests can shorten it, matching pkg/runner's overridable delay.
var cleanupRetryDelay = 500 * time.Millisecond

// cleanupRetryBudget is the wall time one retryTransient call can spend sleeping.
func cleanupRetryBudget() time.Duration {
	var total time.Duration
	for attempt := 0; attempt < cleanupRetryCount-1; attempt++ {
		total += time.Duration(attempt+1) * cleanupRetryDelay
	}
	return total
}

// healthPhaseBudget bounds every per-service health read together. It carries headroom over one
// call's budget on purpose: the point is to stop a many-service suite from paying the allowance
// once per service, not to cut the first service short. A deadline that fires inside the first
// read would report its transient cause as a deadline instead.
//
// The multiplication is worth bounding because client-go retries underneath this loop as well:
// every request defaults to maxRetries 10 and retries a 429 or 5xx that carries a Retry-After
// header, which the API server's priority-and-fairness filter always sets on a 429. Without a
// shared ceiling a many-service teardown could outlive the pod's termination grace period and be
// killed before it ever reaches the warning this file exists to produce.
func healthPhaseBudget() time.Duration {
	return 2 * cleanupRetryBudget()
}

// isTransientCleanupError reports whether a single error is worth another attempt.
func isTransientCleanupError(err error) bool {
	if err == nil {
		return false
	}

	// A cancelled or expired context means the caller is going away, not that the API server is
	// briefly unwell, so it never counts as transient. It is checked first because
	// DeadlineExceeded also satisfies net.Error.
	//
	// This is only safe while env.KubernetesConfig() leaves rest.Config.Timeout at zero. Setting
	// a client timeout there would surface a slow API server as DeadlineExceeded, which this
	// treats as permanent, and the reported symptom would come back.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	if apierrors.IsServiceUnavailable(err) ||
		apierrors.IsTooManyRequests(err) ||
		apierrors.IsServerTimeout(err) ||
		apierrors.IsTimeout(err) {
		return true
	}

	// A connection cut mid-response carries no status, so it has to be matched before the status
	// branch below returns. client-go wraps these as "unexpected error when reading response
	// body. Please retry." Its own wording says what to do with them. A sidecar shutting down
	// at the end of a run produces exactly this.
	//
	// utilnet's helpers compare the sentinels with == and only unwrap url.Error, so they miss
	// the fmt.Errorf wrapping client-go actually applies. errors.Is covers that; the helpers
	// still earn their place for the message-matched cases (GOAWAY, reset, closed connection).
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	if utilnet.IsProbableEOF(err) || utilnet.IsConnectionReset(err) {
		return true
	}

	// A 502 carries the InternalError reason, so the reason-based helpers above miss it. Match it
	// by status code: a gateway error comes from whatever fronts the API server and is as
	// transient as a 503, whereas a plain 500 is usually a failing admission webhook and has to
	// stay loud. A 504 is already covered, because it carries the Timeout reason.
	var status apierrors.APIStatus
	if errors.As(err, &status) {
		return status.Status().Code == http.StatusBadGateway
	}

	// A permanently broken endpoint also satisfies net.Error, so retrying it would only delay a
	// failure the operator has to fix. Name those before the catch-all below.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
		return false
	}
	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) {
		return false
	}
	var unknownAuthority x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthority) {
		return false
	}
	var hostnameErr x509.HostnameError
	if errors.As(err, &hostnameErr) {
		return false
	}

	var netErr net.Error
	return errors.As(err, &netErr)
}

// branches returns the joined errors when err is an aggregate. The group cleanup deletes its
// resource kinds concurrently and joins whatever failed, so most cleanup errors are aggregates,
// even a single failed delete arrives as a one-branch join. Walking the tree is what makes the
// verdict deterministic: the apierrors helpers resolve through errors.As and so read only the
// first status they find, which in a mixed aggregate is whichever goroutine appended first.
//
// An aggregate is recognised only through Unwrap() []error. An apimachinery utilerrors.Aggregate
// does not implement it and would read as one opaque leaf.
func branches(err error) ([]error, bool) {
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		return joined.Unwrap(), true
	}
	return nil, false
}

// cleanupErrorLeaves counts the transient and the total leaves of a cleanup error. Both verdicts
// below are quantifiers over the same walk, so they cannot drift apart.
func cleanupErrorLeaves(err error) (transient, total int) {
	if err == nil {
		return 0, 0
	}
	if parts, ok := branches(err); ok {
		for _, part := range parts {
			partTransient, partTotal := cleanupErrorLeaves(part)
			transient += partTransient
			total += partTotal
		}
		return transient, total
	}
	if isTransientCleanupError(err) {
		return 1, 1
	}
	return 0, 1
}

// shouldRetryCleanupError reports whether another attempt could change the outcome. One transient
// leaf is enough, because the delete calls are idempotent and a later attempt only has to clear
// the one that was failing.
func shouldRetryCleanupError(err error) bool {
	transient, _ := cleanupErrorLeaves(err)
	return transient > 0
}

// isFullyTransientCleanupError reports whether every leaf is transient. This is the stricter
// question the final verdict needs: an aggregate that mixes a permanent failure with a transient
// one must not be downgraded, or a real misconfiguration would be reported only when no blip
// happens to co-occur with it.
func isFullyTransientCleanupError(err error) bool {
	transient, total := cleanupErrorLeaves(err)
	return total > 0 && transient == total
}

// retryTransient calls fn until it succeeds, until no attempt could help, or until the attempts
// run out. A cancelled context stops the loop and joins its cause onto the last error, so the
// caller still sees what actually failed.
func retryTransient(ctx context.Context, count int, delayBase time.Duration, fn func() error) error {
	policy := retry.Policy{Attempts: count, Delay: delayBase, RetryIf: shouldRetryCleanupError}
	return policy.Do(ctx, func(int) error { return fn() })
}
