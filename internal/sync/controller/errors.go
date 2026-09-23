package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	syncagent "github.com/kubeshop/testkube/internal/sync"
)

// terminalOnRejection marks a store rejection that no retry clears as terminal, and returns every
// other error unchanged. The package documentation gives the reasons.
func terminalOnRejection(err error) error {
	if syncagent.IsRejection(err) {
		return reconcile.TerminalError(err)
	}
	return err
}
