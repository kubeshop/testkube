package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

type TestWorkflowStore interface {
	UpdateOrCreateTestWorkflow(context.Context, testworkflowsv1.TestWorkflow) error
	DeleteTestWorkflow(context.Context, string) error
}

func NewTestWorkflowSyncController(mgr ctrl.Manager, store TestWorkflowStore) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&testworkflowsv1.TestWorkflow{}).
		Complete(testWorkflowSyncReconciler(mgr.GetClient(), store)); err != nil {
		return fmt.Errorf("create new sync controller for TestWorkflow: %w", err)
	}
	return nil
}

func testWorkflowSyncReconciler(client client.Reader, store TestWorkflowStore) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		var workflow testworkflowsv1.TestWorkflow
		err := client.Get(ctx, req.NamespacedName, &workflow)
		switch {
		case errors.IsNotFound(err):
			// Deleted, request deletion from store.
			// Passing the name here rather than the namespaced name as generally we refer to objects
			// purely by their name.
			if err := store.DeleteTestWorkflow(ctx, req.Name); err != nil {
				// Unable to delete for some reason, request a retry.
				// We might want to selectively handle different errors here, but ideally they should
				// be handled in the store implementation. If we return abstracted error messages from
				// the store then we should handle them here.
				return ctrl.Result{}, fmt.Errorf("delete TestWorkflow %q from store: %w", req.Name, err)
			}
			return ctrl.Result{}, nil
		case err != nil:
			return ctrl.Result{}, fmt.Errorf("retrieve TestWorkflow %q from Kubernetes: %w", req.NamespacedName, err)
		}

		// Resource has been marked for deletion, we may not get an event when it finally goes so this
		// is the moment when we should update the Control Plane.
		// Kubernetes is a funny thing, when a resource is marked for deletion then the DeletionTimestamp
		// is set, but the resource is not yet removed, giving a chance for controllers to do their thing
		// run finalizers etc. before the resources is removed entirely. Once DeletionTimestamp is set
		// there is no going back so we know this resource is about to be deleted.
		if !workflow.DeletionTimestamp.IsZero() {
			// About to be deleted, request deletion from store.
			// Passing the name here rather than the namespaced name as generally we refer to objects
			// purely by their name.
			if err := store.DeleteTestWorkflow(ctx, req.Name); err != nil {
				// Unable to delete for some reason, request a retry.
				// We might want to selectively handle different errors here, but ideally they should
				// be handled in the store implementation. If we return abstracted error messages from
				// the store then we should handle them here.
				return ctrl.Result{}, fmt.Errorf("delete TestWorkflow %q from store: %w", req.Name, err)
			}
			return ctrl.Result{}, nil
		}

		if hasNoGitOpsSyncAnnotation(&workflow) {
			return ctrl.Result{}, nil
		}

		// Regular update so send the new object into the store.
		if err := store.UpdateOrCreateTestWorkflow(ctx, workflow); err != nil {
			// Unable to update or create for some reason, request a retry.
			// We might want to selectively handle different errors here, but ideally they should
			// be handled in the store implementation. If we return abstracted error messages from
			// the store then we should handle them here.
			return ctrl.Result{}, fmt.Errorf("update TestWorkflow %q in store: %w", workflow.Name, err)
		}

		return ctrl.Result{}, nil
	})
}
