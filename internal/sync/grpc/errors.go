package grpc

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	syncagent "github.com/kubeshop/testkube/internal/sync"
)

// translateError maps Control Plane status codes onto the sentinel errors declared in the parent
// sync package, so that callers can react to the outcome without depending on gRPC.
//
// The Control Plane only uses FailedPrecondition on the sync API to reject a change that conflicts
// with another GitOps agent's ownership of the resource, and InvalidArgument to reject a resource
// that it cannot store. PermissionDenied refuses the Environment of the resource for this agent.
// The result keeps the original error in the chain, because its message names the current owner or
// the invalid value.
func translateError(err error) error {
	if err == nil {
		return nil
	}

	switch status.Code(err) {
	case codes.FailedPrecondition:
		return fmt.Errorf("%w: %w", syncagent.ErrOwnershipConflict, err)
	case codes.InvalidArgument:
		return fmt.Errorf("%w: %w", syncagent.ErrInvalidResource, err)
	case codes.PermissionDenied:
		return fmt.Errorf("%w: %w", syncagent.ErrEnvironmentForbidden, err)
	}

	return err
}
