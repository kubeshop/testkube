package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
)

func TestWebhookSyncReconcilerUpdateOrCreate(t *testing.T) {
	input := executorv1.Webhook{
		Spec: executorv1.WebhookSpec{
			Uri:                "foo",
			Selector:           "bar",
			PayloadObjectField: "baz",
			PayloadTemplate:    "qux",
		},
	}

	store := &fakeStore{}

	reconciler := webhookSyncReconciler(
		fakeKubernetesClient{
			Webhook: input,
		},
		store,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(input, store.Webhook); diff != "" {
		t.Errorf("WebhookSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}
}

func TestWebhookSyncReconcilerDelete(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"

	reconciler := webhookSyncReconciler(
		fakeKubernetesClient{
			Err: fakeNotFoundErr,
		},
		store,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Name: name}}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(name, store.Deleted); diff != "" {
		t.Errorf("WebhookSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}
}
