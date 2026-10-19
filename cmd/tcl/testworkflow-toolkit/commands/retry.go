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
	"errors"
	"net"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const cleanupRetryCount = 5

// cleanupRetryDelay is a var so tests can shorten it, matching pkg/runner's overridable delay.
var cleanupRetryDelay = 500 * time.Millisecond

// isTransientCleanupError walks a joined error, because the group cleanup deletes its resource
// kinds concurrently and joins their errors, while the apierrors helpers read only the first
// status in the tree. Any transient branch makes the whole error worth another attempt.
func isTransientCleanupError(err error) bool {
	if err == nil {
		return false
	}

	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		for _, branch := range joined.Unwrap() {
			if isTransientCleanupError(branch) {
				return true
			}
		}
		return false
	}

	if apierrors.IsServiceUnavailable(err) ||
		apierrors.IsTooManyRequests(err) ||
		apierrors.IsServerTimeout(err) ||
		apierrors.IsTimeout(err) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr)
}

// retryTransient calls fn until it succeeds, returns a non-transient error, or runs out of
// attempts. A cancelled context stops the loop and joins its cause onto the last error, so the
// caller still classifies on what actually failed.
func retryTransient(ctx context.Context, count int, delayBase time.Duration, fn func() error) error {
	var err error
	for attempt := 0; attempt < count; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}
		if !isTransientCleanupError(err) {
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
