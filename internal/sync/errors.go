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

// ErrEnvironmentForbidden is returned by a store when the Control Plane refuses the Environment of
// a resource for this agent. The refusal stands until somebody changes GITOPS_NAMESPACES or the
// Environments of the agent, so callers treat it as terminal.
var ErrEnvironmentForbidden = errors.New("the Control Plane refused the Environment for this agent")

// ErrNameCollision is returned when two watched namespaces that sync into the same Environment hold
// a resource of the same kind and name. The Control Plane keys a resource by its name, so only one
// of them can sync.
var ErrNameCollision = errors.New("another namespace of the same Environment already has a resource with this name")

// IsRejection reports whether err is a rejection that no retry clears: an ownership conflict, an
// invalid resource, a forbidden Environment or a name collision.
func IsRejection(err error) bool {
	return errors.Is(err, ErrOwnershipConflict) ||
		errors.Is(err, ErrInvalidResource) ||
		errors.Is(err, ErrEnvironmentForbidden) ||
		errors.Is(err, ErrNameCollision)
}
