package runner

import (
	"context"
	errors2 "errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/commands"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/artifacts"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/log"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

const (
	GetNotificationsRetryCount = 10
	GetNotificationsRetryDelay = 500 * time.Millisecond

	SaveEndResultRetryCount     = 100
	SaveEndResultRetryBaseDelay = 500 * time.Millisecond

	GetExecutionRetryCount = 200
	GetExecutionRetryDelay = 500 * time.Millisecond

	MonitorRetryCount = 10
	MonitorRetryDelay = 500 * time.Millisecond

	RecoverLogsRetryOnFailureDelay = 300 * time.Millisecond
	RecoverLogsRetryMaxAttempts    = 5

	inlinedGlobalTemplateName = "<inline-global-template>"

	RecoveryRef = "recovery"
)

type RunnerExecute interface {
	Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error)
}

//go:generate mockgen -destination=./mock_runner.go -package=runner "github.com/kubeshop/testkube/pkg/runner" Runner
type Runner interface {
	RunnerExecute
	Monitor(ctx context.Context, organizationId, environmentId, id string) error
	Notifications(ctx context.Context, id string) executionworkertypes.NotificationsWatcher
	Pause(id string) error
	Resume(id string) error
	Abort(id string) error
	Cancel(id string) error
}

type runner struct {
	worker            executionworkertypes.Worker
	client            controlplaneclient.Client
	configRepository  configRepo.Repository
	emitter           event.Interface
	metrics           metrics.Metrics
	proContext        config.ProContext // TODO: Include Agent ID in pro context
	storageSkipVerify bool
	getGlobalTemplate GlobalTemplateFactory

	watching sync.Map
	sf       singleflight.Group
}

func New(
	worker executionworkertypes.Worker,
	configRepository configRepo.Repository,
	client controlplaneclient.Client,
	emitter event.Interface,
	metrics metrics.Metrics,
	proContext config.ProContext,
	storageSkipVerify bool,
	getGlobalTemplate GlobalTemplateFactory,
) Runner {
	return &runner{
		worker:            worker,
		configRepository:  configRepository,
		client:            client,
		emitter:           emitter,
		metrics:           metrics,
		proContext:        proContext,
		storageSkipVerify: storageSkipVerify,
		getGlobalTemplate: getGlobalTemplate,
	}
}

func (r *runner) monitor(ctx context.Context, organizationId string, environmentId string, execution testkube.TestWorkflowExecution) error {
	defer r.watching.Delete(execution.Id)

	var notifications executionworkertypes.NotificationsWatcher
	for i := 0; i < GetNotificationsRetryCount; i++ {
		notifications = r.worker.Notifications(ctx, execution.Id, executionworkertypes.NotificationsOptions{})
		if notifications.Err() == nil {
			break
		}
		if errors.Is(notifications.Err(), registry.ErrResourceNotFound) {
			// TODO: should it mark as job was aborted then?
			return registry.ErrResourceNotFound
		}
		time.Sleep(GetNotificationsRetryDelay)
	}

	if notifications.Err() != nil {
		return errors.Wrapf(notifications.Err(), "failed to listen for '%s' execution notifications", execution.Id)
	}

	logs, err := NewExecutionLogsWriter(r.client, environmentId, execution.Id, execution.Workflow.Name, r.storageSkipVerify)
	if err != nil {
		return err
	}
	saver, err := NewExecutionSaver(ctx, r.client, execution.Id, organizationId, environmentId, r.proContext.Agent.ID, logs, r.proContext.NewArchitecture)
	if err != nil {
		return err
	}
	defer logs.Cleanup()

	type SubRef struct {
		GroupId string
		Name    string
		Index   int
	}
	services := make(map[SubRef]struct{})
	parallel := make(map[SubRef]struct{})
	currentRef := ""
	var lastResult *testkube.TestWorkflowResult
	for n := range notifications.Channel() {
		if n.Output != nil {
			// Track running services
			if n.Output.Name == "service" {
				status, _ := n.Output.Value["status"].(string)
				done, _ := n.Output.Value["done"].(bool)
				groupId, _ := n.Output.Value["group"].(string)
				name, _ := n.Output.Value["name"].(string)
				findex, _ := n.Output.Value["index"].(float64) // JSON marshaler decodes numbers as float64
				index := int(findex)
				if status == string(commands.ServiceStatusQueued) {
					services[SubRef{GroupId: groupId, Name: name, Index: index}] = struct{}{}
				} else if done {
					delete(services, SubRef{GroupId: groupId, Index: index})
				}
			}

			// Track running parallel steps
			if n.Output.Name == "parallel" {
				status, _ := (n.Output.Value["status"]).(string)
				findex, _ := n.Output.Value["index"].(float64) // JSON marshaler decodes numbers as float64
				index := int(findex)
				if status == string(testkube.RUNNING_TestWorkflowStatus) {
					parallel[SubRef{GroupId: n.Output.Ref, Index: index}] = struct{}{}
				} else if testkube.TestWorkflowStatus(status).Finished() {
					delete(parallel, SubRef{GroupId: n.Output.Ref, Index: index})
				}
			}

			// Save the information
			saver.AppendOutput(*n.Output)
		} else if n.Result != nil {
			lastResult = n.Result
			// Don't send final result until everything is finished
			if n.Result.IsFinished() {
				continue
			}
			saver.UpdateResult(*n.Result)
		} else {
			if n.Ref != currentRef && n.Ref != "" {
				currentRef = n.Ref
				err = logs.WriteStart(n.Ref)
				if err != nil {
					log.DefaultLogger.Errorw("failed to write start logs", "id", execution.Id, "ref", n.Ref)
					continue
				}
			}
			_, err = logs.Write([]byte(n.Log))
			if err != nil {
				log.DefaultLogger.Errorw("failed to write logs", "id", execution.Id, "ref", n.Ref)
				continue
			}
		}
	}

	// Ignore further monitoring if it has been canceled
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if notifications.Err() != nil {
		log.DefaultLogger.Errorw("error from TestWorkflow watcher", "id", execution.Id, "error", notifications.Err())
	}

	if lastResult == nil || !lastResult.IsFinished() {
		log.DefaultLogger.Errorw("not finished TestWorkflow result received, trying to recover...", "id", execution.Id)
		watcher := r.worker.Notifications(ctx, execution.Id, executionworkertypes.NotificationsOptions{
			NoFollow: true,
		})
		if watcher.Err() == nil {
			for n := range watcher.Channel() {
				if n.Result != nil {
					lastResult = n.Result
				}
			}
		}

		if lastResult == nil {
			lastResult = execution.Result
		}
		if !lastResult.IsFinished() {
			log.DefaultLogger.Errorw("failed to recover TestWorkflow result, marking as fatal error...", "id", execution.Id)
			lastResult.Fatal(err, true, time.Now())
		}
	}

	// Recover missing data in case the execution has crashed
	if lastResult.IsAborted() || lastResult.IsCanceled() {
		// Service logs
		if len(services) > 0 {
			log.DefaultLogger.Warnw("TestWorkflow execution has been aborted or canceled, while some services are still running. Recovering their logs.", "executionId", execution.Id, "count", len(services))
			for svc := range services {
				err := r.recoverServiceData(ctx, saver, environmentId, &execution, commands.ServiceInfo{
					Group: svc.GroupId,
					Index: int64(svc.Index),
					Name:  svc.Name,
				})
				if err == nil {
					log.DefaultLogger.Infow("recovered TestWorkflow execution service logs", "executionId", execution.Id, "serviceName", svc.Name, "serviceIndex", svc.Index)
				} else {
					log.DefaultLogger.Errorw("failed to recover TestWorkflow execution service logs", "executionId", execution.Id, "serviceName", svc.Name, "serviceIndex", svc.Index, "error", err)
				}
			}
		}

		// Parallel steps logs
		if len(parallel) > 0 {
			log.DefaultLogger.Warnw("TestWorkflow execution has been aborted or canceled, while some parallel steps are still running. Recovering their logs.", "executionId", execution.Id, "count", len(parallel))
			for step := range parallel {
				err := r.recoverParallelStepData(ctx, saver, environmentId, &execution, step.GroupId, int(step.Index))
				if err == nil {
					log.DefaultLogger.Infow("recovered TestWorkflow execution parallel step logs", "executionId", execution.Id, "stepRef", step.GroupId, "stepIndex", step.Index)
				} else {
					log.DefaultLogger.Errorw("failed to recover TestWorkflow execution parallel step logs", "executionId", execution.Id, "stepRef", step.GroupId, "stepIndex", step.Index, "error", err)
				}
			}
		}
	}

	log.DefaultLogger.Infow("Saving execution", "id", execution.Id)
	for i := 0; i < SaveEndResultRetryCount; i++ {
		err = saver.End(ctx, *lastResult)
		if err == nil {
			break
		}
		sleepDuration := time.Duration(i/10) * SaveEndResultRetryBaseDelay
		log.DefaultLogger.Warnw(
			"failed to end execution and save execution data, retrying...",
			"id", execution.Id,
			"retryCount", i,
			"retryDelay", sleepDuration,
			"error", err,
		)
		time.Sleep(sleepDuration)
	}

	// Handle fatal error
	if err != nil {
		log.DefaultLogger.Errorw("failed to save execution data", "id", execution.Id, "error", err)
		return errors.Wrapf(err, "failed to save execution '%s' data", execution.Id)
	}

	// Try to substitute execution data
	execution.Output = nil
	execution.Result = lastResult
	execution.StatusAt = lastResult.FinishedAt

	// Emit data, if the Control Plane doesn't support informing about status by itself
	if !r.proContext.NewArchitecture {
		// Reload latest saved execution to sync data not availabe when monitor was started
		savedExecution, err := r.client.GetExecution(ctx, environmentId, execution.Id)
		if err == nil {
			execution.Signature = savedExecution.Signature
		} else {
			log.DefaultLogger.Errorw("failed to get TestWorkflow execution", "id", execution.Id, "error", err)
		}

		switch {
		case lastResult.IsPassed():
			r.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(&execution))
		case lastResult.IsAborted():
			r.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(&execution))
		case lastResult.IsCanceled():
			r.emitter.Notify(testkube.NewEventEndTestWorkflowCanceled(&execution))
		default:
			r.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(&execution))
		}
		if lastResult.IsNotPassed() {
			r.emitter.Notify(testkube.NewEventEndTestWorkflowNotPassed(&execution))
		}
	}

	err = r.worker.Destroy(context.Background(), execution.Id, executionworkertypes.DestroyOptions{})
	if err != nil {
		// TODO: what to do on error?
		log.DefaultLogger.Errorw("failed to cleanup TestWorkflow resources", "id", execution.Id, "error", err)
	}

	return nil
}

func (r *runner) recoverServiceLogs(ctx context.Context, saver ExecutionSaver, environmentId string, execution *testkube.TestWorkflowExecution, svc commands.ServiceInfo) error {
	storage := artifacts.InternalStorageForAgent(r.client, environmentId, execution.Id, execution.Workflow.Name, RecoveryRef)
	filePath := fmt.Sprintf("logs/%s-%s/%d.log", svc.Group, svc.Name, svc.Index)
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()

	// Load the logs and save as the artifacts
	reader := r.worker.Logs(ctx, fmt.Sprintf("%s-%s-%d", execution.Id, svc.Name, svc.Index), executionworkertypes.LogsOptions{
		Hints:    executionworkertypes.Hints{Namespace: execution.Namespace},
		NoFollow: true,
	})
	if err := reader.Err(); err != nil {
		return err
	}
	if err := storage.SaveStream(filePath, reader); err != nil {
		return err
	}

	// Add information in the execution about the logs
	saver.AppendOutput(testkube.TestWorkflowOutput{
		Ref:  RecoveryRef,
		Name: "service",
		Value: commands.ServiceInfo{
			Group: svc.Group,
			Index: svc.Index,
			Name:  svc.Name,
			Logs:  storage.FullPath(filePath),
			Done:  true,
		}.AsMap(),
	})
	return nil
}

func (r *runner) recoverServiceData(ctx context.Context, saver ExecutionSaver, environmentId string, execution *testkube.TestWorkflowExecution, svc commands.ServiceInfo) (err error) {
	for i := 0; i < RecoverLogsRetryMaxAttempts; i++ {
		if i > 0 {
			// Wait a bit before retrying
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(RecoverLogsRetryOnFailureDelay):
			}
		}

		// Try to recover logs
		if err = r.recoverServiceLogs(ctx, saver, environmentId, execution, svc); err == nil {
			return nil
		}
	}
	return err
}

func (r *runner) recoverParallelStepLogs(ctx context.Context, saver ExecutionSaver, environmentId string, execution *testkube.TestWorkflowExecution, ref string, index int) error {
	storage := artifacts.InternalStorageForAgent(r.client, environmentId, execution.Id, execution.Workflow.Name, RecoveryRef)
	jobName := fmt.Sprintf("%s-%s-%d", execution.Id, ref, index)
	filePath := fmt.Sprintf("logs/%s/%d.log", ref, index)
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()

	// Load the logs and save as the artifacts
	reader := r.worker.Logs(ctx, jobName, executionworkertypes.LogsOptions{
		Hints:    executionworkertypes.Hints{Namespace: execution.Namespace},
		NoFollow: true,
	})
	if err := reader.Err(); err != nil {
		return err
	}
	if err := storage.SaveStream(filePath, reader); err != nil {
		return err
	}
	status := commands.ParallelStatus{
		Index:  index,
		Logs:   storage.FullPath(filePath),
		Status: testkube.ABORTED_TestWorkflowStatus,
	}

	// Load the last acknowledged result of the step and mark it as aborted
	summary, err := r.worker.Get(ctx, jobName, executionworkertypes.GetOptions{
		Hints: executionworkertypes.Hints{Namespace: execution.Namespace},
	})
	if err == nil {
		sigSequence := stage.MapSignatureListToInternal(stage.MapSignatureToSequence(stage.MapSignatureList(summary.Signature)))
		errorMessage := execution.Result.Initialization.ErrorMessage
		if errorMessage == "" {
			for _, sig := range sigSequence {
				if execution.Result.Steps[sig.Ref].ErrorMessage != "" {
					errorMessage = execution.Result.Steps[sig.Ref].ErrorMessage
					break
				}
			}
		}
		status.Result = &summary.Result
		status.Result.Status = common.Ptr(testkube.ABORTED_TestWorkflowStatus)
		status.Result.HealAbortedOrCanceled(sigSequence, errorMessage, controller.DefaultErrorMessage, "aborted")
		status.Result.HealTimestamps(sigSequence, summary.Execution.ScheduledAt, time.Time{}, time.Time{}, true)
		status.Result.HealDuration(summary.Execution.ScheduledAt)
		status.Result.HealMissingPauseStatuses()
		status.Result.HealStatus(sigSequence)
	}

	// Add information in the execution about the logs
	saver.AppendOutput(testkube.TestWorkflowOutput{
		Ref:   ref,
		Name:  "parallel",
		Value: status.AsMap(),
	})
	return nil
}

func (r *runner) recoverParallelStepData(ctx context.Context, saver ExecutionSaver, environmentId string, execution *testkube.TestWorkflowExecution, ref string, index int) (err error) {
	for i := 0; i < RecoverLogsRetryMaxAttempts; i++ {
		if i > 0 {
			// Wait a bit before retrying
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(RecoverLogsRetryOnFailureDelay):
			}
		}

		// Try to recover logs
		if err = r.recoverParallelStepLogs(ctx, saver, environmentId, execution, ref, index); err == nil {
			return nil
		}
	}
	return err
}

func (r *runner) Monitor(ctx context.Context, organizationId string, environmentId string, id string) error {
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()

	// Check if there is any monitor attached already
	r.watching.LoadOrStore(id, false)
	if !r.watching.CompareAndSwap(id, false, true) {
		return nil
	}

	// Load the execution
	var execution *testkube.TestWorkflowExecution
	err := retry(GetExecutionRetryCount, GetExecutionRetryDelay, func(_ int) (err error) {
		execution, err = r.client.GetExecution(ctx, environmentId, id)
		if err != nil {
			log.DefaultLogger.Warnw("failed to get execution for monitoring, retrying...", "id", id, "error", err)
		}
		return err
	})
	if err != nil {
		r.watching.Delete(id)
		log.DefaultLogger.Errorw("failed to get execution for monitoring", "id", id, "error", err)
		return err
	}
	return r.monitor(ctx, organizationId, environmentId, *execution)
}

func (r *runner) Notifications(ctx context.Context, id string) executionworkertypes.NotificationsWatcher {
	return r.worker.Notifications(ctx, id, executionworkertypes.NotificationsOptions{})
}

func (r *runner) Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	v, err, _ := r.sf.Do(request.Execution.Id, func() (interface{}, error) {
		return r.execute(request)
	})
	if err != nil {
		return nil, err
	}
	return v.(*executionworkertypes.ExecuteResult), nil
}

func (r *runner) execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	if request.Execution.OrganizationSlug == "" {
		request.Execution.OrganizationSlug = request.Execution.OrganizationId
	}
	if request.Execution.EnvironmentSlug == "" {
		request.Execution.EnvironmentSlug = request.Execution.EnvironmentId
	}
	if r.getGlobalTemplate != nil {
		globalTemplate, err := r.getGlobalTemplate(request.Execution.EnvironmentId)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get global template")
		}
		testworkflowresolver.AddGlobalTemplateRef(&request.Workflow, testworkflowsv1.TemplateRef{
			Name: testworkflowresolver.GetDisplayTemplateName(inlinedGlobalTemplateName),
		})
		err = testworkflowresolver.ApplyTemplates(&request.Workflow, map[string]*testworkflowsv1.TestWorkflowTemplate{
			inlinedGlobalTemplateName: globalTemplate,
		}, func(key, value string) (expressions.Expression, error) {
			return nil, errors.New("not supported")
		})
		if err != nil {
			return nil, err
		}
	}
	res, err := r.worker.Execute(context.Background(), request)
	if err == nil {
		go func() {
			err := retry(MonitorRetryCount, MonitorRetryDelay, func(_ int) error {
				err := r.Monitor(context.Background(), request.Execution.OrganizationId, request.Execution.EnvironmentId, request.Execution.Id)
				if err != nil {
					log.DefaultLogger.Warnw("failed to monitor execution, retrying...", "id", request.Execution.Id, "error", err)
				}
				return err
			})
			if err != nil {
				log.DefaultLogger.Errorw(
					"failed to monitor execution and retry limit is reached, assuming execution is stuck and running cleanup...",
					"id", request.Execution.Id,
					"error", err,
				)
				// At this point, all retries have failed and nothing is monitoring the execution anymore.
				// We can assume that the execution is stuck in running state and we need to abort it.
				if err := r.abortExecution(context.Background(), request.Execution.EnvironmentId, request.Execution.Id); err != nil {
					log.DefaultLogger.Errorw("failed to abort stuck execution", "id", request.Execution.Id, "error", err)
				}
				log.DefaultLogger.Warnw("aborted execution stuck in running state", "id", request.Execution.Id)
			}
		}()
	}

	// Edge case, when the execution has been already triggered before there,
	// and now it's redundant call.
	if err != nil && strings.Contains(err.Error(), "already exists") {
		existing, existingErr := r.worker.Summary(context.Background(), request.Execution.Id, executionworkertypes.GetOptions{})
		if existingErr != nil {
			return nil, errors2.Join(err, existingErr)
		}
		return &executionworkertypes.ExecuteResult{
			Signature:   existing.Signature,
			ScheduledAt: existing.Execution.ScheduledAt,
			Namespace:   existing.Namespace,
			Redundant:   true,
		}, nil
	}

	return res, err
}

// abortExecution aborts fetches the execution, updates its result to aborted and finishes it.
func (r *runner) abortExecution(ctx context.Context, environmentID, executionID string) error {
	execution, err := r.client.GetExecution(context.Background(), environmentID, executionID)
	if err != nil {
		return errors.Wrapf(err, "failed to get execution '%s'", executionID)
	}
	if execution.Result == nil {
		return errors.New("execution result is nil")
	}
	execution.Result.Fatal(errors.New("execution is stuck in running state"), true, time.Now())
	if err = r.client.UpdateExecutionResult(ctx, environmentID, executionID, execution.Result); err != nil {
		return errors.Wrapf(err, "failed to update execution result '%s' to aborted", executionID)
	}

	if err = r.client.FinishExecutionResult(ctx, environmentID, executionID, execution.Result); err != nil {
		return errors.Wrapf(err, "failed to finish execution result '%s'", executionID)
	}

	if err = r.Abort(executionID); err != nil {
		return errors.Wrapf(err, "failed to destroy execution '%s'", executionID)
	}

	// Emit data, if the Control Plane doesn't support informing about status by itself
	if !r.proContext.NewArchitecture {
		r.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
	}

	return nil
}

func (r *runner) Pause(id string) error {
	return r.worker.Pause(context.Background(), id, executionworkertypes.ControlOptions{})
}

func (r *runner) Resume(id string) error {
	return r.worker.Resume(context.Background(), id, executionworkertypes.ControlOptions{})
}

func (r *runner) Abort(id string) error {
	return r.worker.Abort(context.Background(), id, executionworkertypes.DestroyOptions{})
}

func (r *runner) Cancel(id string) error {
	return r.worker.Cancel(context.Background(), id, executionworkertypes.DestroyOptions{})
}
