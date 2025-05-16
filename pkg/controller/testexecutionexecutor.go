package controller

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testexecutionv1 "github.com/kubeshop/testkube-operator/api/testexecution/v1"
	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	"github.com/kubeshop/testkube/cmd/api-server/services"
	testkubev1 "github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

type TestExecutor = workerpool.Service[testkubev1.Test, testkubev1.ExecutionRequest, testkubev1.Execution]

func NewTestExecutionExecutorController(mgr ctrl.Manager, exec TestExecutor, system *services.DeprecatedSystem) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&testexecutionv1.TestExecution{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(testExecutionExecutor(mgr.GetClient(), exec, system)); err != nil {
		return fmt.Errorf("create new controller for TestExecution: %w", err)
	}
	return nil
}

func testExecutionExecutor(client client.Reader, exec TestExecutor, system *services.DeprecatedSystem) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		// Get and validate the TestExecution.
		var te testexecutionv1.TestExecution
		err := client.Get(ctx, req.NamespacedName, &te)
		switch {
		case errors.IsNotFound(err):
			return ctrl.Result{}, nil
		case err != nil:
			return ctrl.Result{}, err
		case te.Spec.Test == nil:
			return ctrl.Result{}, nil
		case te.Generation == te.Status.Generation:
			return ctrl.Result{}, nil
		case te.Spec.ExecutionRequest == nil:
			te.Spec.ExecutionRequest = &testexecutionv1.ExecutionRequest{}
		}

		test, err := system.Clients.Tests().Get(te.Spec.Test.Name)
		switch {
		case errors.IsNotFound(err):
			return ctrl.Result{}, fmt.Errorf("test does not exist: %w", err)
		case err != nil:
			return ctrl.Result{}, fmt.Errorf("can't get test: %w", err)
		}

		// This is a gross hack but as this controller is legacy it is not worth the effort
		// to map between the types correctly.
		jsonData, err := json.Marshal(te.Spec.ExecutionRequest)
		if err != nil {
			return ctrl.Result{}, err
		}
		var request testkubev1.ExecutionRequest
		if err := json.Unmarshal(jsonData, &request); err != nil {
			return ctrl.Result{}, err
		}

		go exec.SendRequests(system.Scheduler.PrepareTestRequests([]testsv3.Test{*test}, request))
		go exec.Run(ctx)

		log := ctrl.LoggerFrom(ctx)
		log.Info("executed test suite", "name", te.Spec.Test.Name)

		return ctrl.Result{}, nil
	})
}
