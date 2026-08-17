package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
)

type TestTriggerStore interface {
	UpdateOrCreateTestTrigger(context.Context, testtriggersv1.TestTrigger) error
	DeleteTestTrigger(context.Context, string) error
}

func NewTestTriggerSyncController(mgr ctrl.Manager, store TestTriggerStore) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&testtriggersv1.TestTrigger{}).
		Complete(testTriggerSyncReconciler(mgr.GetClient(), mgr.GetEventRecorder("testtrigger-sync-controller"), store)); err != nil {
		return fmt.Errorf("create new sync controller for TestTrigger: %w", err)
	}
	return nil
}

func testTriggerSyncReconciler(client client.Reader, recorder events.EventRecorder, store TestTriggerStore) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		var trigger testtriggersv1.TestTrigger
		err := client.Get(ctx, req.NamespacedName, &trigger)
		switch {
		case errors.IsNotFound(err):
			// Deleted, request deletion from store.
			// Passing the name here rather than the namespaced name as generally we refer to objects
			// purely by their name.
			if err := store.DeleteTestTrigger(ctx, req.Name); err != nil {
				// The resource is already gone from the cluster, so there is nothing left to record
				// an Event against.
				return ctrl.Result{}, reportSyncError(ctx, recorder, nil, fmt.Errorf("delete TestTrigger %q from store: %w", req.Name, err))
			}
			return ctrl.Result{}, nil
		case err != nil:
			return ctrl.Result{}, fmt.Errorf("retrieve TestTrigger %q from Kubernetes: %w", req.NamespacedName, err)
		}

		// Resource has been marked for deletion, we may not get an event when it finally goes so this
		// is the moment when we should update the Control Plane.
		// Kubernetes is a funny thing, when a resource is marked for deletion then the DeletionTimestamp
		// is set, but the resource is not yet removed, giving a chance for controllers to do their thing
		// run finalizers etc. before the resources is removed entirely. Once DeletionTimestamp is set
		// there is no going back so we know this resource is about to be deleted.
		if !trigger.DeletionTimestamp.IsZero() {
			// About to be deleted, request deletion from store.
			// Passing the name here rather than the namespaced name as generally we refer to objects
			// purely by their name.
			if err := store.DeleteTestTrigger(ctx, req.Name); err != nil {
				return ctrl.Result{}, reportSyncError(ctx, recorder, &trigger, fmt.Errorf("delete TestTrigger %q from store: %w", req.Name, err))
			}
			return ctrl.Result{}, nil
		}

		// Regular update so send the new object into the store.
		if err := store.UpdateOrCreateTestTrigger(ctx, trigger); err != nil {
			return ctrl.Result{}, reportSyncError(ctx, recorder, &trigger, fmt.Errorf("update TestTrigger %q in store: %w", trigger.Name, err))
		}

		return ctrl.Result{}, nil
	})
}
