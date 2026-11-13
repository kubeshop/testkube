package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
)

type TestWorkflowTemplateStore interface {
	UpdateOrCreateTestWorkflowTemplate(context.Context, syncagent.Target, testworkflowsv1.TestWorkflowTemplate) error
	DeleteTestWorkflowTemplate(context.Context, syncagent.Target, string) error
}

func NewTestWorkflowTemplateSyncController(mgr ctrl.Manager, store TestWorkflowTemplateStore, target syncagent.TargetFunc) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&testworkflowsv1.TestWorkflowTemplate{}).
		Complete(testWorkflowTemplateSyncReconciler(mgr.GetClient(), store, target)); err != nil {
		return fmt.Errorf("create new sync controller for TestWorkflowTemplate: %w", err)
	}
	return nil
}

func testWorkflowTemplateSyncReconciler(client client.Reader, store TestWorkflowTemplateStore, target syncagent.TargetFunc) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		t := target(req.Namespace)
		var workflowTemplate testworkflowsv1.TestWorkflowTemplate
		err := client.Get(ctx, req.NamespacedName, &workflowTemplate)
		switch {
		case errors.IsNotFound(err):
			// Deleted, request deletion from store.
			if err := store.DeleteTestWorkflowTemplate(ctx, t, req.Name); err != nil {
				return ctrl.Result{}, terminalOnRejection(fmt.Errorf("delete TestWorkflowTemplate %q from store: %w", req.Name, err))
			}
			return ctrl.Result{}, nil
		case err != nil:
			return ctrl.Result{}, fmt.Errorf("retrieve TestWorkflowTemplate %q from Kubernetes: %w", req.NamespacedName, err)
		}

		// Resource has been marked for deletion, we may not get an event when it finally goes so this
		// is the moment when we should update the Control Plane.
		// Kubernetes is a funny thing, when a resource is marked for deletion then the DeletionTimestamp
		// is set, but the resource is not yet removed, giving a chance for controllers to do their thing
		// run finalizers etc. before the resources is removed entirely. Once DeletionTimestamp is set
		// there is no going back so we know this resource is about to be deleted.
		if !workflowTemplate.DeletionTimestamp.IsZero() {
			// About to be deleted, request deletion from store.
			if err := store.DeleteTestWorkflowTemplate(ctx, t, req.Name); err != nil {
				return ctrl.Result{}, terminalOnRejection(fmt.Errorf("delete TestWorkflowTemplate %q from store: %w", req.Name, err))
			}
			return ctrl.Result{}, nil
		}

		// Regular update so send the new object into the store.
		if err := checkNameCollision(ctx, client, &testworkflowsv1.TestWorkflowTemplateList{}, &workflowTemplate, t, target); err != nil {
			return ctrl.Result{}, terminalOnRejection(fmt.Errorf("sync TestWorkflowTemplate %q: %w", req.NamespacedName, err))
		}

		if err := store.UpdateOrCreateTestWorkflowTemplate(ctx, t, workflowTemplate); err != nil {
			return ctrl.Result{}, terminalOnRejection(fmt.Errorf("update TestWorkflowTemplate %q in store: %w", workflowTemplate.Name, err))
		}

		return ctrl.Result{}, nil
	})
}
