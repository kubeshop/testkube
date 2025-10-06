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

	testsuitesv3 "github.com/kubeshop/testkube/api/testsuite/v3"
	testsuiteexecutionv1 "github.com/kubeshop/testkube/api/testsuiteexecution/v1"
	"github.com/kubeshop/testkube/cmd/api-server/services"
	testkubev1 "github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

type TestSuiteExecutor = workerpool.Service[testkubev1.TestSuite, testkubev1.TestSuiteExecutionRequest, testkubev1.TestSuiteExecution]

func NewTestSuiteExecutionExecutorController(mgr ctrl.Manager, exec TestSuiteExecutor, system *services.DeprecatedSystem) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&testsuiteexecutionv1.TestSuiteExecution{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(testSuiteExecutionExecutor(mgr.GetClient(), exec, system)); err != nil {
		return fmt.Errorf("create new controller for TestExecution: %w", err)
	}
	return nil
}

func testSuiteExecutionExecutor(client client.Reader, exec TestSuiteExecutor, system *services.DeprecatedSystem) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		// Get and validate the TestSuiteExecution.
		var tse testsuiteexecutionv1.TestSuiteExecution
		err := client.Get(ctx, req.NamespacedName, &tse)
		switch {
		case errors.IsNotFound(err):
			return ctrl.Result{}, nil
		case err != nil:
			return ctrl.Result{}, err
		case tse.Spec.TestSuite == nil:
			return ctrl.Result{}, nil
		case tse.Generation == tse.Status.Generation:
			return ctrl.Result{}, nil
		case tse.Spec.ExecutionRequest == nil:
			tse.Spec.ExecutionRequest = &testsuiteexecutionv1.TestSuiteExecutionRequest{}
		}

		testSuite, err := system.Clients.TestSuites().Get(tse.Spec.TestSuite.Name)
		switch {
		case errors.IsNotFound(err):
			return ctrl.Result{}, fmt.Errorf("test suite does not exist: %w", err)
		case err != nil:
			return ctrl.Result{}, fmt.Errorf("can't get test suite: %w", err)
		}

		tse.Spec.ExecutionRequest.RunningContext = &testsuiteexecutionv1.RunningContext{
			Type_:   testsuiteexecutionv1.RunningContextTypeTestSuiteExecution,
			Context: tse.Name,
		}

		// This is a gross hack but as this controller is legacy it is not worth the effort
		// to map between the types correctly.
		jsonData, err := json.Marshal(tse.Spec.ExecutionRequest)
		if err != nil {
			return ctrl.Result{}, err
		}
		var request testkubev1.TestSuiteExecutionRequest
		if err := json.Unmarshal(jsonData, &request); err != nil {
			return ctrl.Result{}, err
		}

		go exec.SendRequests(system.Scheduler.PrepareTestSuiteRequests([]testsuitesv3.TestSuite{*testSuite}, request))
		go exec.Run(ctx)

		log := ctrl.LoggerFrom(ctx)
		log.Info("executed test suite", "name", tse.Spec.TestSuite.Name)

		return ctrl.Result{}, nil
	})
}
