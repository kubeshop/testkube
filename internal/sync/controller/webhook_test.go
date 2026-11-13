package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
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
		testTarget,
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
		testTarget,
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
		testTarget,
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

func TestWebhookSyncReconcilerTarget(t *testing.T) {
	want := syncagent.Target{Namespace: "team-b", EnvironmentID: "env-b"}
	tests := []struct {
		name   string
		client fakeKubernetesClient
	}{
		{
			name:   "the target of an update comes from the namespace of the request",
			client: fakeKubernetesClient{Webhook: executorv1.Webhook{ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "team-b"}}},
		},
		{
			name:   "the target of a delete comes from the namespace of the request",
			client: fakeKubernetesClient{Err: fakeNotFoundErr},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			reconciler := webhookSyncReconciler(tt.client, store, testTarget)

			_, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "team-b", Name: "smoke"}})

			if err != nil {
				t.Fatalf("reconciliation failed: %v", err)
			}
			if diff := cmp.Diff(want, store.Target); diff != "" {
				t.Errorf("TestWebhookSyncReconcilerTarget: -want, +got:\n%s", diff)
			}
		})
	}
}
