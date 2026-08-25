package controller

import (
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	syncagent "github.com/kubeshop/testkube/internal/sync"
)

// terminalOnOwnershipConflict decides whether a failed store operation is worth retrying.
//
// An ownership conflict stands until somebody reassigns the owner or removes the resource from this
// agent's scope, so retrying only produces noise. Those are marked terminal to stop the requeue
// loop. Every other failure is returned unchanged so that it is retried with backoff.
//
// Nothing is logged here: controller-runtime logs whatever a reconciler returns, terminal or not,
// along with the kind and name of the resource, and the Control Plane's message naming the current
// owner travels along inside err.
func terminalOnOwnershipConflict(err error) error {
	if errors.Is(err, syncagent.ErrOwnershipConflict) {
		return reconcile.TerminalError(err)
	}
	return err
}
