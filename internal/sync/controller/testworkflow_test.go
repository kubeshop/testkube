package controller

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
)

func TestTestWorkflowSyncReconcilerUpdateOrCreate(t *testing.T) {
	input := testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{
			TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
				Config: map[string]testworkflowsv1.ParameterSchema{
					"foo": {
						Description: "foo",
						Type:        "bar",
					},
					"baz": {
						Description: "baz",
						Type:        "qux",
					},
				},
				Concurrency: &testworkflowsv1.ConcurrencyPolicy{
					Group: "foo",
					Max:   5,
				},
			},
		},
	}

	store := &fakeStore{}

	reconciler := testWorkflowSyncReconciler(
		fakeKubernetesClient{
			TestWorkflow: input,
		},
		store,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(input, store.TestWorkflow); diff != "" {
		t.Errorf("TestWorkflowSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 1 {
		t.Errorf("TestWorkflowSyncReconcilerUpdateOrCreate: expected 1 update call, got %d", store.UpdateCalls)
	}
}

func TestTestWorkflowSyncReconcilerDelete(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"

	reconciler := testWorkflowSyncReconciler(
		fakeKubernetesClient{
			Err: fakeNotFoundErr,
		},
		store,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Name: name}}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(name, store.Deleted); diff != "" {
		t.Errorf("TestWorkflowSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 0 {
		t.Errorf("TestWorkflowSyncReconcilerDelete: expected 0 update calls, got %d", store.UpdateCalls)
	}
}

func TestTestWorkflowSyncReconcilerDeleteWhenMarkedForDeletion(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"
	now := metav1.Now()

	reconciler := testWorkflowSyncReconciler(
		fakeKubernetesClient{
			TestWorkflow: testworkflowsv1.TestWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:              name,
					DeletionTimestamp: &now,
				},
			},
		},
		store,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Name: name}}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(name, store.Deleted); diff != "" {
		t.Errorf("TestWorkflowSyncReconcilerDeleteWhenMarkedForDeletion: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 0 {
		t.Errorf("TestWorkflowSyncReconcilerDeleteWhenMarkedForDeletion: expected 0 update calls, got %d", store.UpdateCalls)
	}
}

// When the Control Plane rejects a sync because another GitOps agent owns the resource, the
// reconciler must give up rather than requeue the same doomed request forever.
func TestTestWorkflowSyncReconcilerOwnershipConflictIsTerminal(t *testing.T) {
	name := "smoke"
	store := &fakeStore{Err: fmt.Errorf("rejected: %w", syncagent.ErrOwnershipConflict)}

	reconciler := testWorkflowSyncReconciler(
		fakeKubernetesClient{
			TestWorkflow: testworkflowsv1.TestWorkflow{
				ObjectMeta: metav1.ObjectMeta{Name: name},
			},
		},
		store,
	)

	_, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Name: name}})

	if !errors.Is(err, reconcile.TerminalError(nil)) {
		t.Errorf("expected a terminal error, got %v", err)
	}
	if !errors.Is(err, syncagent.ErrOwnershipConflict) {
		t.Errorf("expected the ownership conflict to stay in the error chain, got %v", err)
	}
}
