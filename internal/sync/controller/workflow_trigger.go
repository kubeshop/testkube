package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	workflowtriggersv1 "github.com/kubeshop/testkube/api/workflowtriggers/v1"
	syncagent "github.com/kubeshop/testkube/internal/sync"
)

type WorkflowTriggerStore interface {
	UpdateOrCreateWorkflowTrigger(context.Context, syncagent.Target, workflowtriggersv1.WorkflowTrigger) error
	DeleteWorkflowTrigger(context.Context, syncagent.Target, string) error
}

func NewWorkflowTriggerSyncController(mgr ctrl.Manager, store WorkflowTriggerStore, resolveTarget syncagent.TargetFunc) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&workflowtriggersv1.WorkflowTrigger{}).
		Complete(workflowTriggerSyncReconciler(mgr.GetClient(), store, resolveTarget)); err != nil {
		return fmt.Errorf("create new sync controller for WorkflowTrigger: %w", err)
	}
	return nil
}

func workflowTriggerSyncReconciler(client client.Reader, store WorkflowTriggerStore, resolveTarget syncagent.TargetFunc) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		target := resolveTarget(req.Namespace)
		var trigger workflowtriggersv1.WorkflowTrigger
		err := client.Get(ctx, req.NamespacedName, &trigger)
		switch {
		case errors.IsNotFound(err):
			if err := store.DeleteWorkflowTrigger(ctx, target, req.Name); err != nil {
				return ctrl.Result{}, terminalOnRejection(fmt.Errorf("delete WorkflowTrigger %q from store: %w", req.Name, err))
			}
			return ctrl.Result{}, nil
		case err != nil:
			return ctrl.Result{}, fmt.Errorf("retrieve WorkflowTrigger %q from Kubernetes: %w", req.NamespacedName, err)
		}

		if !trigger.DeletionTimestamp.IsZero() {
			if err := store.DeleteWorkflowTrigger(ctx, target, req.Name); err != nil {
				return ctrl.Result{}, terminalOnRejection(fmt.Errorf("delete WorkflowTrigger %q from store: %w", req.Name, err))
			}
			return ctrl.Result{}, nil
		}

		if err := checkNameCollision(ctx, client, &workflowtriggersv1.WorkflowTriggerList{}, &trigger, target, resolveTarget); err != nil {
			return ctrl.Result{}, terminalOnRejection(fmt.Errorf("sync WorkflowTrigger %q: %w", req.NamespacedName, err))
		}

		if err := store.UpdateOrCreateWorkflowTrigger(ctx, target, trigger); err != nil {
			return ctrl.Result{}, terminalOnRejection(fmt.Errorf("update WorkflowTrigger %q in store: %w", trigger.Name, err))
		}

		return ctrl.Result{}, nil
	})
}
