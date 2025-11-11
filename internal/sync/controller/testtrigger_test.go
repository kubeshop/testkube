package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
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
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(input, store.TestTrigger); diff != "" {
		t.Errorf("TestTriggerSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
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
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Name: name}}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(name, store.Deleted); diff != "" {
		t.Errorf("TestTriggerSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}
}
