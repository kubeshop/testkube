package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
)

func TestWorkflowTriggerSyncReconcilerUpdateOrCreate(t *testing.T) {
	input := workflowtriggersv1.WorkflowTrigger{
		Spec: workflowtriggersv1.WorkflowTriggerSpec{
			When: workflowtriggersv1.WorkflowTriggerWhen{Event: "modified"},
			Run: workflowtriggersv1.WorkflowTriggerRun{
				Workflow: workflowtriggersv1.WorkflowTriggerWorkflowSelector{Name: "smoke-test"},
			},
		},
	}

	store := &fakeStore{}

	reconciler := workflowTriggerSyncReconciler(
		fakeKubernetesClient{
			WorkflowTrigger: input,
		},
		store,
		testTarget,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(input, store.WorkflowTrigger); diff != "" {
		t.Errorf("WorkflowTriggerSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 1 {
		t.Errorf("WorkflowTriggerSyncReconcilerUpdateOrCreate: expected 1 update call, got %d", store.UpdateCalls)
	}
}

func TestWorkflowTriggerSyncReconcilerDelete(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"

	reconciler := workflowTriggerSyncReconciler(
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
		t.Errorf("WorkflowTriggerSyncReconcilerDelete: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 0 {
		t.Errorf("WorkflowTriggerSyncReconcilerDelete: expected 0 update calls, got %d", store.UpdateCalls)
	}
}

func TestWorkflowTriggerSyncReconcilerDeleteWhenMarkedForDeletion(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"
	now := metav1.Now()

	reconciler := workflowTriggerSyncReconciler(
		fakeKubernetesClient{
			WorkflowTrigger: workflowtriggersv1.WorkflowTrigger{
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
		t.Errorf("WorkflowTriggerSyncReconcilerDeleteWhenMarkedForDeletion: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 0 {
		t.Errorf("WorkflowTriggerSyncReconcilerDeleteWhenMarkedForDeletion: expected 0 update calls, got %d", store.UpdateCalls)
	}
}

// The reconciler works out the target of a resource from the namespace of the request, for the
// update and for the delete.
func TestWorkflowTriggerSyncReconcilerTarget(t *testing.T) {
	want := syncagent.Target{Namespace: "team-b", EnvironmentID: "env-b"}
	tests := []struct {
		name   string
		client fakeKubernetesClient
	}{
		{
			name:   "the target of an update comes from the namespace of the request",
			client: fakeKubernetesClient{WorkflowTrigger: workflowtriggersv1.WorkflowTrigger{ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "team-b"}}},
		},
		{
			name:   "the target of a delete comes from the namespace of the request",
			client: fakeKubernetesClient{Err: fakeNotFoundErr},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			reconciler := workflowTriggerSyncReconciler(tt.client, store, testTarget)

			_, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "team-b", Name: "smoke"}})

			if err != nil {
				t.Fatalf("reconciliation failed: %v", err)
			}
			if diff := cmp.Diff(want, store.Target); diff != "" {
				t.Errorf("TestWorkflowTriggerSyncReconcilerTarget: -want, +got:\n%s", diff)
			}
		})
	}
}
