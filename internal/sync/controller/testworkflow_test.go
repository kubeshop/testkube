package controller

import (
	"errors"
	"fmt"
	"testing"
	"time"

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
		testTarget,
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
		testTarget,
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
		testTarget,
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
		testTarget,
	)

	_, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Name: name}})

	if !errors.Is(err, reconcile.TerminalError(nil)) {
		t.Errorf("expected a terminal error, got %v", err)
	}
	if !errors.Is(err, syncagent.ErrOwnershipConflict) {
		t.Errorf("expected the ownership conflict to stay in the error chain, got %v", err)
	}
}

// The reconciler works out the target of a resource from the namespace of the request, for the
// update and for the delete.
func TestTestWorkflowSyncReconcilerTarget(t *testing.T) {
	want := syncagent.Target{Namespace: "team-b", EnvironmentID: "env-b"}
	tests := []struct {
		name   string
		client fakeKubernetesClient
	}{
		{
			name:   "the target of an update comes from the namespace of the request",
			client: fakeKubernetesClient{TestWorkflow: testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "team-b"}}},
		},
		{
			name:   "the target of a delete comes from the namespace of the request",
			client: fakeKubernetesClient{Err: fakeNotFoundErr},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			reconciler := testWorkflowSyncReconciler(tt.client, store, testTarget)

			_, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "team-b", Name: "smoke"}})

			if err != nil {
				t.Fatalf("reconciliation failed: %v", err)
			}
			if diff := cmp.Diff(want, store.Target); diff != "" {
				t.Errorf("TestTestWorkflowSyncReconcilerTarget: -want, +got:\n%s", diff)
			}
		})
	}
}

// A second TestWorkflow with the same name in a namespace of the same Environment never reaches
// the Control Plane, and the reconciler does not retry it.
func TestTestWorkflowSyncReconcilerNameCollisionIsTerminal(t *testing.T) {
	earlier := metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	later := metav1.NewTime(earlier.Add(time.Hour))
	winner := testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "team-b", CreationTimestamp: earlier}}
	loser := testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "team-c", CreationTimestamp: later}}
	store := &fakeStore{}

	reconciler := testWorkflowSyncReconciler(
		fakeKubernetesClient{
			TestWorkflow:  loser,
			TestWorkflows: []testworkflowsv1.TestWorkflow{winner, loser},
		},
		store,
		testTarget,
	)

	_, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "team-c", Name: "smoke"}})

	if !errors.Is(err, reconcile.TerminalError(nil)) {
		t.Errorf("expected a terminal error, got %v", err)
	}
	if !errors.Is(err, syncagent.ErrNameCollision) {
		t.Errorf("expected the name collision to stay in the error chain, got %v", err)
	}
	if store.UpdateCalls != 0 {
		t.Errorf("expected 0 update calls, got %d", store.UpdateCalls)
	}
}
