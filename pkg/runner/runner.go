package runner

import (
	"context"
	errors2 "errors"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/log"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

const (
	GetNotificationsRetryCount = 10
	GetNotificationsRetryDelay = 500 * time.Millisecond

	SaveEndResultRetryCount     = 100
	SaveEndResultRetryBaseDelay = 500 * time.Millisecond

	inlinedGlobalTemplateName = "<inline-global-template>"
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

	currentRef := ""
	var lastResult *testkube.TestWorkflowResult
	for n := range notifications.Channel() {
		if n.Temporary {
			continue
		}

		if n.Output != nil {
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

	for i := 0; i < SaveEndResultRetryCount; i++ {
		err = saver.End(ctx, *lastResult)
		if err == nil {
			break
		}
		log.DefaultLogger.Warnw("failed to save execution data", "id", execution.Id, "error", err)
		time.Sleep(time.Duration(i/10) * SaveEndResultRetryBaseDelay)
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
		if lastResult.IsPassed() {
			r.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(&execution))
		} else if lastResult.IsAborted() {
			r.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(&execution))
		} else {
			r.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(&execution))
		}
	}

	err = r.worker.Destroy(context.Background(), execution.Id, executionworkertypes.DestroyOptions{})
	if err != nil {
		// TODO: what to do on error?
		log.DefaultLogger.Errorw("failed to cleanup TestWorkflow resources", "id", execution.Id, "error", err)
	}

	return nil
}

func (r *runner) Monitor(ctx context.Context, organizationId string, environmentId string, id string) error {
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()

	// Check if there is any monitor attached already
	r.watching.LoadOrStore(id, false)
	if !r.watching.CompareAndSwap(id, false, true) {
		return nil
	}

	// Load the execution TODO: retry?
	execution, err := r.client.GetExecution(ctx, environmentId, id)
	if err != nil {
		log.DefaultLogger.Errorw("failed to get execution", "id", id, "error", err)
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
		// TODO: consider retry?
		go func() {
			for i := 0; i < 3; i++ {
				err := r.Monitor(context.Background(), request.Execution.OrganizationId, request.Execution.EnvironmentId, request.Execution.Id)
				if err == nil {
					return
				}
				log.DefaultLogger.Errorw("failed to monitor execution", "id", request.Execution.Id, "error", err)
				time.Sleep(1 * time.Second)
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

func (r *runner) Pause(id string) error {
	return r.worker.Pause(context.Background(), id, executionworkertypes.ControlOptions{})
}

func (r *runner) Resume(id string) error {
	return r.worker.Resume(context.Background(), id, executionworkertypes.ControlOptions{})
}

func (r *runner) Abort(id string) error {
	return r.worker.Abort(context.Background(), id, executionworkertypes.DestroyOptions{})
}
