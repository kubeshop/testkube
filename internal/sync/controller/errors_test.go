package controller

import (
	"errors"
	"fmt"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	syncagent "github.com/kubeshop/testkube/internal/sync"
)

// An ownership conflict cannot be resolved by retrying, so it has to come back as a terminal error
// to stop controller-runtime requeueing it forever.
func TestOwnershipConflictIsTerminal(t *testing.T) {
	err := fmt.Errorf("update TestWorkflow %q in store: %w", "smoke", syncagent.ErrOwnershipConflict)

	got := terminalOnOwnershipConflict(err)

	if !errors.Is(got, reconcile.TerminalError(nil)) {
		t.Errorf("expected a terminal error so that controller-runtime stops requeueing, got %v", got)
	}
	// The Control Plane names the current owner in its message, and that message is the only clue
	// the user gets from the agent log.
	if !errors.Is(got, syncagent.ErrOwnershipConflict) {
		t.Errorf("expected the ownership conflict to stay in the error chain, got %v", got)
	}
}

// Every other failure is transient as far as the agent can tell, so it has to stay retryable.
func TestOtherErrorsStayRetryable(t *testing.T) {
	err := errors.New("connection refused")

	got := terminalOnOwnershipConflict(err)

	if errors.Is(got, reconcile.TerminalError(nil)) {
		t.Errorf("expected a retryable error, got terminal error %v", got)
	}
	if !errors.Is(got, err) {
		t.Errorf("expected the original error to be returned unchanged, got %v", got)
	}
}
