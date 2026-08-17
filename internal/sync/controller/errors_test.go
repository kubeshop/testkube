package controller

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
)

// An ownership conflict cannot be resolved by retrying, so it must come back as a terminal error
// and be recorded against the resource for the user to find with `kubectl describe`.
func TestReportSyncErrorMarksOwnershipConflictTerminal(t *testing.T) {
	recorder := events.NewFakeRecorder(1)
	workflow := testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "team-a"}}
	err := fmt.Errorf("update TestWorkflow %q in store: %w", "smoke", syncagent.ErrOwnershipConflict)

	got := reportSyncError(t.Context(), recorder, &workflow, err)

	if !errors.Is(got, reconcile.TerminalError(nil)) {
		t.Errorf("expected a terminal error so that controller-runtime stops requeueing, got %v", got)
	}
	if !errors.Is(got, syncagent.ErrOwnershipConflict) {
		t.Errorf("expected the ownership conflict to stay in the error chain, got %v", got)
	}

	select {
	case event := <-recorder.Events:
		if !strings.Contains(event, ownershipConflictReason) {
			t.Errorf("expected event to carry reason %q, got %q", ownershipConflictReason, event)
		}
	default:
		t.Error("expected a Warning event to be recorded against the resource, got none")
	}
}

// Every other failure is transient as far as the agent can tell, so it must stay retryable and not
// generate Event noise.
func TestReportSyncErrorLeavesOtherErrorsRetryable(t *testing.T) {
	recorder := events.NewFakeRecorder(1)
	err := errors.New("connection refused")

	got := reportSyncError(t.Context(), recorder, &testworkflowsv1.TestWorkflow{}, err)

	if errors.Is(got, reconcile.TerminalError(nil)) {
		t.Errorf("expected a retryable error, got terminal error %v", got)
	}
	if !errors.Is(got, err) {
		t.Errorf("expected the original error to be returned unchanged, got %v", got)
	}

	select {
	case event := <-recorder.Events:
		t.Errorf("expected no event for a retryable error, got %q", event)
	default:
	}
}

// A conflict reported while deleting has no resource left to attach an Event to, but it still must
// not be retried.
func TestReportSyncErrorWithoutResourceStillTerminal(t *testing.T) {
	recorder := events.NewFakeRecorder(1)
	err := fmt.Errorf("delete TestWorkflow %q from store: %w", "smoke", syncagent.ErrOwnershipConflict)

	got := reportSyncError(t.Context(), recorder, nil, err)

	if !errors.Is(got, reconcile.TerminalError(nil)) {
		t.Errorf("expected a terminal error, got %v", got)
	}

	select {
	case event := <-recorder.Events:
		t.Errorf("expected no event when there is no resource to record against, got %q", event)
	default:
	}
}
