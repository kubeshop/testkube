package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	errors2 "github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/event"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/repository/channels"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

const (
	apiKeyMeta  = "api-key"
	agentIdMeta = "agent-id"
	orgIdMeta   = "organization-id"
	envIdMeta   = "environment-id"
)

const (
	saveResultRetryMaxAttempts = 100
	saveResultRetryBaseDelay   = 300 * time.Millisecond
)

type agentLoop struct {
	runner              Runner
	worker              executionworkertypes.Worker
	logger              *zap.SugaredLogger
	emitter             event.Interface
	client              controlplaneclient.Client
	proContext          config.ProContext
	agentId             string
	organizationId      string
	legacyEnvironmentId string

	newExecutionsEnabled bool
}

type AgentLoop interface {
	Start(ctx context.Context) error
}

func newAgentLoop(
	runner Runner,
	worker executionworkertypes.Worker,
	logger *zap.SugaredLogger,
	emitter event.Interface,
	client controlplaneclient.Client,
	proContext config.ProContext,
	agentId string,
	organizationId string,
	legacyEnvironmentId string,
	newExecutionsEnabled bool,
) AgentLoop {
	return &agentLoop{
		runner:               runner,
		worker:               worker,
		logger:               logger,
		emitter:              emitter,
		client:               client,
		proContext:           proContext,
		agentId:              agentId,
		organizationId:       organizationId,
		legacyEnvironmentId:  legacyEnvironmentId,
		newExecutionsEnabled: newExecutionsEnabled,
	}
}

func (a *agentLoop) Start(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := a.run(ctx)

		a.logger.Errorw("runner agent connection failed, reconnecting", "error", err)

		// TODO: some smart back off strategy?
		time.Sleep(5 * time.Second)
	}
}

func (a *agentLoop) _saveEmptyLogs(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	workflowName := ""
	if execution.Workflow != nil {
		workflowName = execution.Workflow.Name
	}
	return a.client.SaveExecutionLogs(ctx, environmentId, execution.Id, workflowName, nil)
}

func (a *agentLoop) saveEmptyLogs(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	err := retry(saveResultRetryMaxAttempts, saveResultRetryBaseDelay, func() error {
		return a._saveEmptyLogs(ctx, environmentId, execution)
	})
	if err != nil {
		a.logger.Errorw("failed to save empty log", "executionId", execution.Id, "error", err)
	}
	return err
}

func (a *agentLoop) finishExecution(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	err := retry(saveResultRetryMaxAttempts, saveResultRetryBaseDelay, func() error {
		err := a.client.FinishExecutionResult(ctx, environmentId, execution.Id, execution.Result)
		if err != nil {
			a.logger.Warnw("failed to finish the TestWorkflow execution in database", "recoverable", true, "executionId", execution.Id, "error", err)
			return err
		}
		if !a.newExecutionsEnabled {
			// Emit events locally if the Control Plane doesn't support that
			//a.emitter.Notify(testkube.NewEventStartTestWorkflow(execution)) // TODO: delete - sent from Cloud
			if execution.Result.IsPassed() {
				a.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(execution))
			} else if execution.Result.IsAborted() {
				a.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
			} else {
				a.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(execution))
			}
		}
		return nil
	})
	if err != nil {
		a.logger.Errorw("failed to finish the TestWorkflow execution in database", "recoverable", false, "executionId", execution.Id, "error", err)
	}
	return err
}

func (a *agentLoop) init(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	err := retry(saveResultRetryMaxAttempts, saveResultRetryBaseDelay, func() (err error) {
		err = a.client.InitExecution(ctx, environmentId, execution.Id, execution.Signature, execution.Namespace)
		if err != nil {
			a.logger.Warnw("failed to initialize the TestWorkflow execution in database", "recoverable", true, "executionId", execution.Id, "error", err)
		}
		return err
	})
	if err != nil {
		a.logger.Errorw("failed to initialize the TestWorkflow execution in database", "recoverable", false, "executionId", execution.Id, "error", err)
	}
	return err
}

func (a *agentLoop) run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	// Handle the new mechanism for runners
	if a.newExecutionsEnabled {
		g.Go(func() error {
			return errors2.Wrap(a.loopRunnerRequests(ctx), "runners loop")
		})
	}

	// Handle Test Workflow notifications of all kinds
	g.Go(func() error {
		return errors2.Wrap(a.loopNotifications(ctx), "notifications loop")
	})
	g.Go(func() error {
		return errors2.Wrap(a.loopServiceNotifications(ctx), "service notifications loop")
	})
	g.Go(func() error {
		return errors2.Wrap(a.loopParallelStepNotifications(ctx), "parallel steps notifications loop")
	})

	return g.Wait()
}

func (a *agentLoop) loopNotifications(ctx context.Context) error {
	return a.client.ProcessExecutionNotificationRequests(ctx, func(ctx context.Context, req *cloud.TestWorkflowNotificationsRequest) channels.Watcher[*testkube.TestWorkflowExecutionNotification] {
		// Read the initial status TODO: consider getting from the database
		status, err := a.worker.Summary(ctx, req.ExecutionId, executionworkertypes.GetOptions{})
		if err != nil {
			return channels.NewError[*testkube.TestWorkflowExecutionNotification](err)
		}

		// Fail fast when it's already finished
		if status.EstimatedResult.IsFinished() {
			return channels.NewError[*testkube.TestWorkflowExecutionNotification](fmt.Errorf("failed to fetch real-time notifications: execution is already finished"))
		}

		// Start reading the notifications
		return a.worker.Notifications(ctx, status.Resource.Id, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{
				Namespace:   status.Namespace,
				Signature:   status.Signature,
				ScheduledAt: common.Ptr(status.Execution.ScheduledAt),
			},
		})
	})
}

func (a *agentLoop) loopServiceNotifications(ctx context.Context) error {
	return a.client.ProcessExecutionServiceNotificationRequests(ctx, func(ctx context.Context, req *cloud.TestWorkflowServiceNotificationsRequest) channels.Watcher[*testkube.TestWorkflowExecutionNotification] {
		// Build the internal resource name
		resourceId := fmt.Sprintf("%s-%s-%d", req.ExecutionId, req.ServiceName, req.ServiceIndex)

		// Start reading the notifications
		return a.worker.Notifications(ctx, resourceId, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{},
		})
	})
}

func (a *agentLoop) loopParallelStepNotifications(ctx context.Context) error {
	return a.client.ProcessExecutionParallelWorkerNotificationRequests(ctx, func(ctx context.Context, req *cloud.TestWorkflowParallelStepNotificationsRequest) channels.Watcher[*testkube.TestWorkflowExecutionNotification] {
		// Build the internal resource name
		resourceId := fmt.Sprintf("%s-%s-%d", req.ExecutionId, req.Ref, req.WorkerIndex)

		// Start reading the notifications
		return a.worker.Notifications(ctx, resourceId, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{},
		})
	})
}

func (a *agentLoop) loopRunnerRequests(ctx context.Context) error {
	watcher := a.client.WatchRunnerRequests(ctx)
	for req := range watcher.Channel() {
		go func(req *cloud.RunnerRequest) {
			// Lock the execution for itself
			var resp *cloud.ObtainExecutionResponse
			resp, err := a.client.ObtainExecution(ctx, req.EnvironmentId, req.Id)
			if err != nil {
				a.logger.Errorf("failed to obtain execution '%s/%s', from Control Plane: %v", req.EnvironmentId, req.Id, err)
				return
			}

			// Ignore if the resource has been locked before
			if !resp.Success {
				return
			}

			// Continue
			err = a.runTestWorkflow(req.EnvironmentId, req.Id, resp.Token)
			if err != nil {
				a.logger.Errorf("failed to run execution '%s/%s' from Control Plane: %v", req.EnvironmentId, req.Id, err)
			}
		}(req)
	}
	return watcher.Err()
}

func (a *agentLoop) runTestWorkflow(environmentId string, executionId string, executionToken string) error {
	ctx := context.Background()
	logger := a.logger.With("environmentId", environmentId, "executionId", executionId)

	// Get the execution details
	execution, err := a.client.GetExecution(ctx, environmentId, executionId)
	if err != nil {
		return errors2.Wrapf(err, "failed to get execution details '%s/%s' from Control Plane", environmentId, executionId)
	}

	// TODO: Pass it there?
	controlPlaneConfig := testworkflowconfig.ControlPlaneConfig{
		// TODO
		//DashboardUrl:   e.dashboardURI,
		//CDEventsTarget: e.cdEventsTarget,
	}

	parentIds := ""
	if execution.RunningContext != nil && execution.RunningContext.Actor != nil {
		parentIds = execution.RunningContext.Actor.ExecutionPath
	}
	result, err := a.runner.Execute(executionworkertypes.ExecuteRequest{
		Token: executionToken,
		Execution: testworkflowconfig.ExecutionConfig{
			Id:              execution.Id,
			GroupId:         execution.GroupId,
			Name:            execution.Name,
			Number:          execution.Number,
			ScheduledAt:     execution.ScheduledAt,
			DisableWebhooks: execution.DisableWebhooks,
			Debug:           false,
			OrganizationId:  a.organizationId,
			EnvironmentId:   environmentId,
			ParentIds:       parentIds,
		},
		Workflow:     testworkflowmappers.MapTestWorkflowAPIToKube(*execution.ResolvedWorkflow),
		ControlPlane: controlPlaneConfig,
	})

	// TODO: define "revoke" error by runner (?)
	if err != nil {
		execution.InitializationError("Failed to run execution", err)
		_ = a.saveEmptyLogs(context.Background(), environmentId, execution)
		err2 := a.finishExecution(context.Background(), environmentId, execution)
		err = errors.Join(err, err2)
		if err != nil {
			logger.Errorw("failed to run and update execution", "error", err)
		}
		return nil
	}

	// Inform about execution start
	//e.emitter.Notify(testkube.NewEventStartTestWorkflow(execution)) // TODO: delete - sent from Cloud

	// Apply the known data to temporary object.
	execution.Namespace = result.Namespace
	execution.Signature = result.Signature
	execution.RunnerId = a.agentId
	if err = a.init(context.Background(), environmentId, execution); err != nil {
		logger.Errorw("failed to mark execution as initialized", "error", err)
	}

	return nil
}
