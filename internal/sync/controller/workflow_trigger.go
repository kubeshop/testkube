package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
)

type WorkflowTriggerStore interface {
	UpdateOrCreateWorkflowTrigger(context.Context, workflowtriggersv1.WorkflowTrigger) error
	DeleteWorkflowTrigger(context.Context, string) error
}

func NewWorkflowTriggerSyncController(mgr ctrl.Manager, store WorkflowTriggerStore) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&workflowtriggersv1.WorkflowTrigger{}).
		Complete(workflowTriggerSyncReconciler(mgr.GetClient(), store)); err != nil {
		return fmt.Errorf("create new sync controller for WorkflowTrigger: %w", err)
	}
	return nil
}

func workflowTriggerSyncReconciler(client client.Reader, store WorkflowTriggerStore) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		var trigger workflowtriggersv1.WorkflowTrigger
		err := client.Get(ctx, req.NamespacedName, &trigger)
		switch {
		case errors.IsNotFound(err):
			if err := store.DeleteWorkflowTrigger(ctx, req.Name); err != nil {
				return ctrl.Result{}, fmt.Errorf("delete WorkflowTrigger %q from store: %w", req.Name, err)
			}
			return ctrl.Result{}, nil
		case err != nil:
			return ctrl.Result{}, fmt.Errorf("retrieve WorkflowTrigger %q from Kubernetes: %w", req.NamespacedName, err)
		}

		if !trigger.DeletionTimestamp.IsZero() {
			if err := store.DeleteWorkflowTrigger(ctx, req.Name); err != nil {
				return ctrl.Result{}, fmt.Errorf("delete WorkflowTrigger %q from store: %w", req.Name, err)
			}
			return ctrl.Result{}, nil
		}

		if err := store.UpdateOrCreateWorkflowTrigger(ctx, trigger); err != nil {
			return ctrl.Result{}, fmt.Errorf("update WorkflowTrigger %q in store: %w", trigger.Name, err)
		}

		return ctrl.Result{}, nil
	})
}
