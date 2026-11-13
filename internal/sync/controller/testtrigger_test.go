package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
)

func TestTestTriggerSyncReconcilerUpdateOrCreate(t *testing.T) {
	input := testtriggersv1.TestTrigger{
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:  "foo",
			Event:     "bar",
			Action:    "baz",
			Execution: "qux",
		},
	}

	store := &fakeStore{}

	reconciler := testTriggerSyncReconciler(
		fakeKubernetesClient{
			TestTrigger: input,
		},
		store,
		testTarget,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(input, store.TestTrigger); diff != "" {
		t.Errorf("TestTriggerSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 1 {
		t.Errorf("TestTriggerSyncReconcilerUpdateOrCreate: expected 1 update call, got %d", store.UpdateCalls)
	}
}

func TestTestTriggerSyncReconcilerDelete(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"

	reconciler := testTriggerSyncReconciler(
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
		t.Errorf("TestTriggerSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 0 {
		t.Errorf("TestTriggerSyncReconcilerDelete: expected 0 update calls, got %d", store.UpdateCalls)
	}
}

func TestTestTriggerSyncReconcilerDeleteWhenMarkedForDeletion(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"
	now := metav1.Now()

	reconciler := testTriggerSyncReconciler(
		fakeKubernetesClient{
			TestTrigger: testtriggersv1.TestTrigger{
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
		t.Errorf("TestTriggerSyncReconcilerDeleteWhenMarkedForDeletion: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 0 {
		t.Errorf("TestTriggerSyncReconcilerDeleteWhenMarkedForDeletion: expected 0 update calls, got %d", store.UpdateCalls)
	}
}

func TestTestTriggerSyncReconcilerTarget(t *testing.T) {
	want := syncagent.Target{Namespace: "team-b", EnvironmentID: "env-b"}
	tests := []struct {
		name   string
		client fakeKubernetesClient
	}{
		{
			name:   "the target of an update comes from the namespace of the request",
			client: fakeKubernetesClient{TestTrigger: testtriggersv1.TestTrigger{ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "team-b"}}},
		},
		{
			name:   "the target of a delete comes from the namespace of the request",
			client: fakeKubernetesClient{Err: fakeNotFoundErr},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			reconciler := testTriggerSyncReconciler(tt.client, store, testTarget)

			_, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "team-b", Name: "smoke"}})

			if err != nil {
				t.Fatalf("reconciliation failed: %v", err)
			}
			if diff := cmp.Diff(want, store.Target); diff != "" {
				t.Errorf("TestTestTriggerSyncReconcilerTarget: -want, +got:\n%s", diff)
			}
		})
	}
}
