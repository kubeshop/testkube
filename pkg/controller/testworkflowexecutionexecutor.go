package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	testkubev1 "github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

type TestWorkflowExecutor interface {
	Execute(ctx context.Context, request *cloud.ScheduleRequest) ([]testkubev1.TestWorkflowExecution, error)
}

func NewTestWorkflowExecutionExecutorController(mgr ctrl.Manager, exec TestWorkflowExecutor) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&testworkflowsv1.TestWorkflowExecution{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(testWorkflowExecutionExecutor(mgr.GetClient(), mgr.GetEventRecorderFor("testworkflowexecution-controller"), exec)); err != nil {
		return fmt.Errorf("create new controller for TestWorkflowExecution: %w", err)
	}
	return nil
}

func testWorkflowExecutionExecutor(client client.Client, recorder record.EventRecorder, exec TestWorkflowExecutor) reconcile.Reconciler {
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

		_, err = exec.Execute(ctx, &cloud.ScheduleRequest{
			Executions:           []*cloud.ScheduleExecution{&scheduleExecution},
			DisableWebhooks:      twe.Spec.ExecutionRequest.DisableWebhooks,
			Tags:                 twe.Spec.ExecutionRequest.Tags,
			RunningContext:       rc,
			KubernetesObjectName: twe.Spec.ExecutionRequest.TestWorkflowExecutionName,
			User:                 user,
		})

		if err != nil {
			execErr := fmt.Errorf("executing test workflow from execution %q: %w", twe.Name, err)
			// Update the status with the error and set Generation to prevent infinite retries.
			var statusTWE testworkflowsv1.TestWorkflowExecution
			if getErr := client.Get(ctx, req.NamespacedName, &statusTWE); getErr != nil {
				return ctrl.Result{}, fmt.Errorf("getting fresh execution %q for error status update: %w (original error: %w)", twe.Name, getErr, execErr)
			}
			statusTWE.Status.Generation = twe.Generation
			statusTWE.Status.Error = err.Error()
			if updateErr := client.Status().Update(ctx, &statusTWE); updateErr != nil {
				return ctrl.Result{}, fmt.Errorf("updating error status for execution %q: %w (original error: %w)", twe.Name, updateErr, execErr)
			}
			recorder.Event(&statusTWE, corev1.EventTypeWarning, "ExecutionNotScheduled", err.Error())
			return ctrl.Result{}, nil
		}

		// Update the status generation to prevent re-execution on operator restart.
		var statusTWE testworkflowsv1.TestWorkflowExecution
		if err := client.Get(ctx, req.NamespacedName, &statusTWE); err != nil {
			return ctrl.Result{}, fmt.Errorf("getting fresh execution %q for status update: %w", twe.Name, err)
		}
		statusTWE.Status.Generation = twe.Generation
		statusTWE.Status.Error = ""
		if err := client.Status().Update(ctx, &statusTWE); err != nil {
			return ctrl.Result{}, fmt.Errorf("updating status generation for execution %q: %w", twe.Name, err)
		}

		recorder.Event(&statusTWE, corev1.EventTypeNormal, "ExecutionScheduled", fmt.Sprintf("Scheduled test workflow %q", twe.Spec.TestWorkflow.Name))

		log := ctrl.LoggerFrom(ctx)
		log.Info("executed test workflow", "name", twe.Spec.TestWorkflow.Name)

		return ctrl.Result{}, nil
	})
}
