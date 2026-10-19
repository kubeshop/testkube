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
	"net"
	"net/http"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const cleanupRetryCount = 5

// cleanupRetryDelay is a var so tests can shorten it, matching pkg/runner's overridable delay.
var cleanupRetryDelay = 500 * time.Millisecond

// isTransientCleanupError reports whether a single error is worth another attempt.
func isTransientCleanupError(err error) bool {
	if err == nil {
		return false
	}

	// A cancelled or expired context means the caller is going away, not that the API server is
	// briefly unwell, so it never counts as transient. It is checked first because
	// DeadlineExceeded also satisfies net.Error.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	if apierrors.IsServiceUnavailable(err) ||
		apierrors.IsTooManyRequests(err) ||
		apierrors.IsServerTimeout(err) ||
		apierrors.IsTimeout(err) {
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
// resource kinds concurrently and joins whatever failed, so most cleanup errors are aggregates.
func branches(err error) ([]error, bool) {
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		return joined.Unwrap(), true
	}
	return nil, false
}

// shouldRetryCleanupError reports whether another attempt could change the outcome. One transient
// branch is enough, because the delete calls are idempotent and a later attempt only has to clear
// the branch that was failing.
func shouldRetryCleanupError(err error) bool {
	if err == nil {
		return false
	}
	if parts, ok := branches(err); ok {
		for _, part := range parts {
			if shouldRetryCleanupError(part) {
				return true
			}
		}
		return false
	}
	return isTransientCleanupError(err)
}

// isFullyTransientCleanupError reports whether every branch is transient. This is the stricter
// question the final verdict needs: an aggregate that mixes a permanent failure with a transient
// one must not be downgraded, or a real misconfiguration would be reported only when no blip
// happens to co-occur with it.
func isFullyTransientCleanupError(err error) bool {
	if err == nil {
		return false
	}
	if parts, ok := branches(err); ok {
		if len(parts) == 0 {
			return false
		}
		for _, part := range parts {
			if !isFullyTransientCleanupError(part) {
				return false
			}
		}
		return true
	}
	return isTransientCleanupError(err)
}

// retryTransient calls fn until it succeeds, until no attempt could help, or until the attempts
// run out. A cancelled context stops the loop and joins its cause onto the last error, so the
// caller still sees what actually failed.
func retryTransient(ctx context.Context, count int, delayBase time.Duration, fn func() error) error {
	var err error
	for attempt := 0; attempt < count; attempt++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			if err == nil {
				return ctxErr
			}
			return errors.Join(err, ctxErr)
		}

		err = fn()
		if err == nil {
			return nil
		}
		if !shouldRetryCleanupError(err) {
			return err
		}
		if attempt == count-1 {
			break
		}

		select {
		case <-ctx.Done():
			return errors.Join(err, ctx.Err())
		case <-time.After(time.Duration(attempt+1) * delayBase):
		}
	}
	return err
}
