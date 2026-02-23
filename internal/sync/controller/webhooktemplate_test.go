package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if store.UpdateCalls != 1 {
		t.Errorf("WebhookTemplateSyncReconcilerUpdateOrCreate: expected 1 update call, got %d", store.UpdateCalls)
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

	if store.UpdateCalls != 0 {
		t.Errorf("WebhookTemplateSyncReconcilerDelete: expected 0 update calls, got %d", store.UpdateCalls)
	}
}

func TestWebhookTemplateSyncReconcilerDeleteWhenMarkedForDeletion(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"
	now := metav1.Now()

	reconciler := webhookTemplateSyncReconciler(
		fakeKubernetesClient{
			WebhookTemplate: executorv1.WebhookTemplate{
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
		t.Errorf("WebhookTemplateSyncReconcilerDeleteWhenMarkedForDeletion: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 0 {
		t.Errorf("WebhookTemplateSyncReconcilerDeleteWhenMarkedForDeletion: expected 0 update calls, got %d", store.UpdateCalls)
	}
}

func TestWebhookTemplateSyncReconcilerSkipsWhenNoGitOpsSyncAnnotationIsSet(t *testing.T) {
	store := &fakeStore{}

	reconciler := webhookTemplateSyncReconciler(
		fakeKubernetesClient{
			WebhookTemplate: executorv1.WebhookTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						noGitOpsSyncAnnotation: "true",
					},
				},
			},
		},
		store,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if store.UpdateCalls != 0 {
		t.Errorf("TestWebhookTemplateSyncReconcilerSkipsWhenNoGitOpsSyncAnnotationIsSet: expected 0 update calls, got %d", store.UpdateCalls)
	}

	if store.Deleted != "" {
		t.Errorf("TestWebhookTemplateSyncReconcilerSkipsWhenNoGitOpsSyncAnnotationIsSet: expected no delete call, got %q", store.Deleted)
	}
}
