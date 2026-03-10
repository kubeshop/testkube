package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if store.UpdateCalls != 1 {
		t.Errorf("WebhookSyncReconcilerUpdateOrCreate: expected 1 update call, got %d", store.UpdateCalls)
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

	if store.UpdateCalls != 0 {
		t.Errorf("WebhookSyncReconcilerDelete: expected 0 update calls, got %d", store.UpdateCalls)
	}
}

func TestWebhookSyncReconcilerDeleteWhenMarkedForDeletion(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"
	now := metav1.Now()

	reconciler := webhookSyncReconciler(
		fakeKubernetesClient{
			Webhook: executorv1.Webhook{
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
		t.Errorf("WebhookSyncReconcilerDeleteWhenMarkedForDeletion: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 0 {
		t.Errorf("WebhookSyncReconcilerDeleteWhenMarkedForDeletion: expected 0 update calls, got %d", store.UpdateCalls)
	}
}
