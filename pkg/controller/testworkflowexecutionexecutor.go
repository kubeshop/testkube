package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	testkubev1 "github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

type TestWorkflowExecutor interface {
	Execute(ctx context.Context, namespace string, request *cloud.ScheduleRequest) testworkflowexecutor.TestWorkflowExecutionStream
}

func NewTestWorkflowExecutionExecutorController(mgr ctrl.Manager, exec TestWorkflowExecutor) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&testworkflowsv1.TestWorkflowExecution{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(testWorkflowExecutionExecutor(mgr.GetClient(), exec)); err != nil {
		return fmt.Errorf("create new controller for TestWorkflowExecution: %w", err)
	}
	return nil
}

func testWorkflowExecutionExecutor(client client.Reader, exec TestWorkflowExecutor) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		// Get and validate the TestWorkflowExecution.
		var twe testworkflowsv1.TestWorkflowExecution
		err := client.Get(ctx, req.NamespacedName, &twe)
		switch {
		case errors.IsNotFound(err):
			return ctrl.Result{}, nil
		case err != nil:
			return ctrl.Result{}, err
		case twe.Spec.TestWorkflow == nil:
			return ctrl.Result{}, nil
		case twe.Generation == twe.Status.Generation:
			return ctrl.Result{}, nil
		case twe.Spec.ExecutionRequest == nil:
			twe.Spec.ExecutionRequest = &testworkflowsv1.TestWorkflowExecutionRequest{}
		}

		// Wrangle the Kubernetes type into the internal representation used by the executor.
		interface_ := testkubev1.API_TestWorkflowRunningContextInterfaceType
		actor := testkubev1.TESTWORKFLOWEXECUTION_TestWorkflowRunningContextActorType
		runningContext := &testkubev1.TestWorkflowRunningContext{
			Interface_: &testkubev1.TestWorkflowRunningContextInterface{
				Type_: &interface_,
			},
			Actor: &testkubev1.TestWorkflowRunningContextActor{
				Name:  twe.Name,
				Type_: &actor,
			},
		}
		rc, user := testworkflowexecutor.GetNewRunningContext(runningContext, nil)

		var scheduleExecution cloud.ScheduleExecution
		if twe.Spec.ExecutionRequest.Target != nil {
			target := &cloud.ExecutionTarget{Replicate: twe.Spec.ExecutionRequest.Target.Replicate}
			if twe.Spec.ExecutionRequest.Target.Match != nil {
				target.Match = make(map[string]*cloud.ExecutionTargetLabels)
				for k, v := range twe.Spec.ExecutionRequest.Target.Match {
					target.Match[k] = &cloud.ExecutionTargetLabels{Labels: v}
				}
			}
			if twe.Spec.ExecutionRequest.Target.Not != nil {
				target.Not = make(map[string]*cloud.ExecutionTargetLabels)
				for k, v := range twe.Spec.ExecutionRequest.Target.Not {
					target.Not[k] = &cloud.ExecutionTargetLabels{Labels: v}
				}
			}
			scheduleExecution.Targets = []*cloud.ExecutionTarget{target}
		}
		scheduleExecution.Selector = &cloud.ScheduleResourceSelector{Name: twe.Spec.TestWorkflow.Name}
		scheduleExecution.Config = testworkflows.MapConfigValueKubeToAPI(twe.Spec.ExecutionRequest.Config)

		executions := exec.Execute(ctx, "", &cloud.ScheduleRequest{
			Executions:           []*cloud.ScheduleExecution{&scheduleExecution},
			DisableWebhooks:      twe.Spec.ExecutionRequest.DisableWebhooks,
			Tags:                 twe.Spec.ExecutionRequest.Tags,
			RunningContext:       rc,
			KubernetesObjectName: twe.Spec.ExecutionRequest.TestWorkflowExecutionName,
			User:                 user,
		})

		if executions.Error() != nil {
			return ctrl.Result{}, fmt.Errorf("executing test workflow from execution %q: %w", twe.Name, executions.Error())
		}

		log := ctrl.LoggerFrom(ctx)
		log.Info("executed test workflow", "name", twe.Spec.TestWorkflow.Name)

		return ctrl.Result{}, nil
	})
}
