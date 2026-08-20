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
// with another GitOps agent's ownership of the resource. The original error is kept in the chain
// because its message names the current owner.
func translateError(err error) error {
	if err == nil {
		return nil
	}

	if status.Code(err) == codes.FailedPrecondition {
		return fmt.Errorf("%w: %w", syncagent.ErrOwnershipConflict, err)
	}

	return err
}
