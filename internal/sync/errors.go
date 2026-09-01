package sync

import "errors"

// ErrOwnershipConflict is returned by a store when the Control Plane rejects a change because the
// resource is owned by a different GitOps agent. Retrying cannot resolve this, so callers should
// treat it as terminal and surface it to the user instead of requeueing.
//
// Ownership is reassigned by setting the testkube.io/gitops-owner annotation on the Kubernetes
// resource, or by removing the resource from the owning agent's scope.
var ErrOwnershipConflict = errors.New("resource is owned by another GitOps agent")
