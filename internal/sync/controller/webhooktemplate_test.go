package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
)

func TestWebhookTemplateSyncReconcilerUpdateOrCreate(t *testing.T) {
	input := executorv1.WebhookTemplate{
		Spec: executorv1.WebhookTemplateSpec{
			Uri:                "foo",
			Selector:           "bar",
			PayloadObjectField: "baz",
			PayloadTemplate:    "qux",
		},
	}

	store := &fakeStore{}

	reconciler := webhookTemplateSyncReconciler(
		fakeKubernetesClient{
			WebhookTemplate: input,
		},
		store,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(input, store.WebhookTemplate); diff != "" {
		t.Errorf("WebhookTemplateSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}
}

func TestWebhookTemplateSyncReconcilerDelete(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"

	reconciler := webhookTemplateSyncReconciler(
		fakeKubernetesClient{
			Err: fakeNotFoundErr,
		},
		store,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Name: name}}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(name, store.Deleted); diff != "" {
		t.Errorf("WebhookTemplateSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}
}
