package sync

import "errors"

// ErrOwnershipConflict is returned by a store when the Control Plane rejects a change because the
// resource is owned by a different GitOps agent. Retrying cannot resolve this, so callers should
// treat it as terminal and surface it to the user instead of requeueing.
//
// Ownership is reassigned by setting the testkube.io/gitops-owner annotation on the Kubernetes
// resource, or by removing the resource from the owning agent's scope.
var ErrOwnershipConflict = errors.New("resource is owned by another GitOps agent")

// ErrInvalidResource is returned by a store when the Control Plane rejects a resource as invalid,
// for example a webhook condition with an unknown type. The Control Plane rejects the same resource
// on every retry, so callers treat it as terminal until somebody changes the resource.
var ErrInvalidResource = errors.New("the Control Plane rejected the resource as invalid")

// IsRejection reports whether the Control Plane rejected a resource in a way that no retry clears:
// an ownership conflict or an invalid resource.
func IsRejection(err error) bool {
	return errors.Is(err, ErrOwnershipConflict) || errors.Is(err, ErrInvalidResource)
}
