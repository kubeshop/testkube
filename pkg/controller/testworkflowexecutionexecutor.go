package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Complete(testWorkflowExecutionExecutor(mgr.GetClient(), exec)); err != nil {
		return fmt.Errorf("create new controller for TestWorkflowExecution: %w", err)
	}
	return nil
}

func testWorkflowExecutionExecutor(k8sClient client.Client, exec TestWorkflowExecutor) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
		log := ctrl.LoggerFrom(ctx)

		// Get and validate the TestWorkflowExecution.
		var twe testworkflowsv1.TestWorkflowExecution
		err := k8sClient.Get(ctx, req.NamespacedName, &twe)
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

		// Execute the workflow - use twe.Name as the KubernetesObjectName so that
		// the event listener can find this CRD and update its status.
		executions, err := exec.Execute(ctx, &cloud.ScheduleRequest{
			Executions:           []*cloud.ScheduleExecution{&scheduleExecution},
			DisableWebhooks:      twe.Spec.ExecutionRequest.DisableWebhooks,
			Tags:                 twe.Spec.ExecutionRequest.Tags,
			RunningContext:       rc,
			KubernetesObjectName: twe.Name,
			User:                 user,
		})

		if err != nil {
			// Update status to reflect the scheduling error
			if updateErr := updateStatusWithError(ctx, k8sClient, &twe, err); updateErr != nil {
				log.Error(updateErr, "failed to update TestWorkflowExecution status with error")
			}
			return ctrl.Result{}, fmt.Errorf("executing test workflow from execution %q: %w", twe.Name, err)
		}

		// Update the status with the initial execution details.
		// Subsequent status updates (START, END events) will be handled by the
		// testworkflowexecutions event listener.
		if len(executions) > 0 {
			if updateErr := updateStatusWithExecution(ctx, k8sClient, &twe, &executions[0]); updateErr != nil {
				log.Error(updateErr, "failed to update TestWorkflowExecution status")
				// Don't return error - the execution was scheduled successfully
			}
		}

		log.Info("executed test workflow", "name", twe.Spec.TestWorkflow.Name, "executionCount", len(executions))
		return ctrl.Result{}, nil
	})
}

// updateStatusWithExecution updates the TestWorkflowExecution CRD status with the execution details.
func updateStatusWithExecution(ctx context.Context, k8sClient client.Client, twe *testworkflowsv1.TestWorkflowExecution, execution *testkubev1.TestWorkflowExecution) error {
	twe.Status = testworkflows.MapTestWorkflowExecutionStatusAPIToKube(execution, twe.Generation)
	return k8sClient.Status().Update(ctx, twe)
}

// updateStatusWithError updates the TestWorkflowExecution CRD status to reflect a scheduling failure.
func updateStatusWithError(ctx context.Context, k8sClient client.Client, twe *testworkflowsv1.TestWorkflowExecution, err error) error {
	now := metav1.NewTime(time.Now())
	failedStatus := testworkflowsv1.FAILED_TestWorkflowStatus
	failedStepStatus := testworkflowsv1.FAILED_TestWorkflowStepStatus

	twe.Status = testworkflowsv1.TestWorkflowExecutionStatus{
		LatestExecution: &testworkflowsv1.TestWorkflowExecutionDetails{
			Name:        twe.Name,
			Namespace:   twe.Namespace,
			ScheduledAt: now,
			StatusAt:    now,
			Result: &testworkflowsv1.TestWorkflowResult{
				Status:          &failedStatus,
				PredictedStatus: &failedStatus,
				QueuedAt:        now,
				FinishedAt:      now,
				Initialization: &testworkflowsv1.TestWorkflowStepResult{
					Status:       &failedStepStatus,
					ErrorMessage: fmt.Sprintf("Failed to schedule execution: %s", err.Error()),
					QueuedAt:     now,
					FinishedAt:   now,
				},
				Steps: map[string]testworkflowsv1.TestWorkflowStepResult{},
			},
		},
		Generation: twe.Generation,
	}
	return k8sClient.Status().Update(ctx, twe)
}
