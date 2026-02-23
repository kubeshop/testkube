package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

func TestTestWorkflowTemplateSyncReconcilerUpdateOrCreate(t *testing.T) {
	input := testworkflowsv1.TestWorkflowTemplate{
		Spec: testworkflowsv1.TestWorkflowTemplateSpec{
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

	reconciler := testWorkflowTemplateSyncReconciler(
		fakeKubernetesClient{
			TestWorkflowTemplate: input,
		},
		store,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(input, store.TestWorkflowTemplate); diff != "" {
		t.Errorf("TestWorkflowTemplateSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 1 {
		t.Errorf("TestWorkflowTemplateSyncReconcilerUpdateOrCreate: expected 1 update call, got %d", store.UpdateCalls)
	}
}

func TestTestWorkflowTemplateSyncReconcilerDelete(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"

	reconciler := testWorkflowTemplateSyncReconciler(
		fakeKubernetesClient{
			Err: fakeNotFoundErr,
		},
		store,
	)

	if _, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: types.NamespacedName{Name: name}}); err != nil {
		t.Errorf("reconciliation failed: %v", err)
	}

	if diff := cmp.Diff(name, store.Deleted); diff != "" {
		t.Errorf("TestWorkflowTemplateSyncReconcilerUpdateOrCreate: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 0 {
		t.Errorf("TestWorkflowTemplateSyncReconcilerDelete: expected 0 update calls, got %d", store.UpdateCalls)
	}
}

func TestTestWorkflowTemplateSyncReconcilerDeleteWhenMarkedForDeletion(t *testing.T) {
	store := &fakeStore{}
	name := "foobar"
	now := metav1.Now()

	reconciler := testWorkflowTemplateSyncReconciler(
		fakeKubernetesClient{
			TestWorkflowTemplate: testworkflowsv1.TestWorkflowTemplate{
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
		t.Errorf("TestWorkflowTemplateSyncReconcilerDeleteWhenMarkedForDeletion: -want, +got:\n%s", diff)
	}

	if store.UpdateCalls != 0 {
		t.Errorf("TestWorkflowTemplateSyncReconcilerDeleteWhenMarkedForDeletion: expected 0 update calls, got %d", store.UpdateCalls)
	}
}

func TestTestWorkflowTemplateSyncReconcilerSkipsWhenNoGitOpsSyncAnnotationIsSet(t *testing.T) {
	store := &fakeStore{}

	reconciler := testWorkflowTemplateSyncReconciler(
		fakeKubernetesClient{
			TestWorkflowTemplate: testworkflowsv1.TestWorkflowTemplate{
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
		t.Errorf("TestTestWorkflowTemplateSyncReconcilerSkipsWhenNoGitOpsSyncAnnotationIsSet: expected 0 update calls, got %d", store.UpdateCalls)
	}

	if store.Deleted != "" {
		t.Errorf("TestTestWorkflowTemplateSyncReconcilerSkipsWhenNoGitOpsSyncAnnotationIsSet: expected no delete call, got %q", store.Deleted)
	}
}
