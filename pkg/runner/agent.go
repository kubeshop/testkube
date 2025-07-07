package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	errors2 "github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

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
	saveResultRetryMaxAttempts = 100
	saveResultRetryBaseDelay   = 300 * time.Millisecond
	agentLoopReconnectionDelay = 3 * time.Second
)

type agentLoop struct {
	runner              Runner
	worker              executionworkertypes.Worker
	logger              *zap.SugaredLogger
	emitter             event.Interface
	client              controlplaneclient.Client
	proContext          config.ProContext
	controlPlaneConfig  testworkflowconfig.ControlPlaneConfig
	organizationId      string
	legacyEnvironmentId string
	sf                  singleflight.Group
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
	controlPlaneConfig testworkflowconfig.ControlPlaneConfig,
	proContext config.ProContext,
	organizationId string,
	legacyEnvironmentId string,
) AgentLoop {
	return &agentLoop{
		runner:              runner,
		worker:              worker,
		logger:              logger,
		emitter:             emitter,
		client:              client,
		proContext:          proContext,
		controlPlaneConfig:  controlPlaneConfig,
		organizationId:      organizationId,
		legacyEnvironmentId: legacyEnvironmentId,
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
		time.Sleep(agentLoopReconnectionDelay)
	}
}

func (a *agentLoop) _saveEmptyLogs(ctx context.Context, environmentId string, execution *testkube.TestWorkflowExecution) error {
	workflowName := ""
	if execution.Workflow != nil {
		workflowName = execution.Workflow.Name
	}
	return a.client.SaveExecutionLogs(ctx, environmentId, execution.Id, workflowName, strings.NewReader(""))
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
		if !a.proContext.NewArchitecture {
			switch {
			case execution.Result.IsPassed():
				a.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(execution))
			case execution.Result.IsAborted():
				a.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
			case execution.Result.IsCanceled():
				a.emitter.Notify(testkube.NewEventEndTestWorkflowCanceled(execution))
			default:
				a.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(execution))
			}
			if execution.Result.IsNotPassed() {
				a.emitter.Notify(testkube.NewEventEndTestWorkflowNotPassed(execution))
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
	if a.proContext.NewArchitecture {
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
	return a.client.ProcessExecutionNotificationRequests(ctx, func(ctx context.Context, req *cloud.TestWorkflowNotificationsRequest) controlplaneclient.NotificationWatcher {
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
	return a.client.ProcessExecutionServiceNotificationRequests(ctx, func(ctx context.Context, req *cloud.TestWorkflowServiceNotificationsRequest) controlplaneclient.NotificationWatcher {
		// Build the internal resource name
		resourceId := fmt.Sprintf("%s-%s-%d", req.ExecutionId, req.ServiceName, req.ServiceIndex)

		// Start reading the notifications
		return a.worker.Notifications(ctx, resourceId, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{},
		})
	})
}

func (a *agentLoop) loopParallelStepNotifications(ctx context.Context) error {
	return a.client.ProcessExecutionParallelWorkerNotificationRequests(ctx, func(ctx context.Context, req *cloud.TestWorkflowParallelStepNotificationsRequest) controlplaneclient.NotificationWatcher {
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
	var wg sync.WaitGroup
	for req := range watcher.Channel() {
		wg.Add(1)
		go func(req controlplaneclient.RunnerRequest) {
			defer wg.Done()
			switch req.Type() {
			case cloud.RunnerRequestType_CONSIDER:
				if err := req.Consider().Send(&cloud.RunnerConsiderResponse{Ok: true}); err != nil {
					a.logger.Errorf("failed to accept the '%s/%s' execution: %v", req.EnvironmentID(), req.ExecutionID(), err)
					return
				}
			case cloud.RunnerRequestType_START:
				err := a.runTestWorkflow(req.EnvironmentID(), req.ExecutionID(), req.Start().Token())
				if err == nil {
					err = req.Start().Send(&cloud.RunnerStartResponse{})
					if err != nil {
						a.logger.Errorf("failed to send success for start execution '%s/%s': %v", req.EnvironmentID(), req.ExecutionID(), err)
					}
				} else {
					a.logger.Errorf("failed to start execution '%s/%s': %v", req.EnvironmentID(), req.ExecutionID(), err)
					err = req.Start().SendError(err)
					if err != nil {
						a.logger.Errorf("failed to send error for start execution '%s/%s': %v", req.EnvironmentID(), req.ExecutionID(), err)
					}
				}
			case cloud.RunnerRequestType_CANCEL:
				a.logger.Infow("received cancel request for execution", "environmentId", req.EnvironmentID(), "executionId", req.ExecutionID())
				originalError := a.runner.Cancel(req.ExecutionID())
				if originalError != nil {
					err := req.SendError(originalError)
					if err != nil {
						a.logger.Errorf("failed to send cancel '%s/%s' error: %v: %v", req.EnvironmentID(), req.ExecutionID(), originalError, err)
					}
				} else {
					err := req.Cancel().Send()
					if err != nil {
						a.logger.Errorf("failed to send cancel '%s/%s' success: %v", req.EnvironmentID(), req.ExecutionID(), err)
					}
				}
			case cloud.RunnerRequestType_ABORT:
				originalError := a.runner.Abort(req.ExecutionID())
				if originalError != nil {
					err := req.SendError(originalError)
					if err != nil {
						a.logger.Errorf("failed to send abort '%s/%s' error: %v: %v", req.EnvironmentID(), req.ExecutionID(), originalError, err)
					}
				} else {
					err := req.Abort().Send()
					if err != nil {
						a.logger.Errorf("failed to send abort '%s/%s' success: %v", req.EnvironmentID(), req.ExecutionID(), err)
					}
				}
			case cloud.RunnerRequestType_PAUSE:
				originalError := a.runner.Pause(req.ExecutionID())
				if originalError != nil {
					err := req.SendError(originalError)
					if err != nil {
						a.logger.Errorf("failed to send pause '%s/%s' error: %v: %v", req.EnvironmentID(), req.ExecutionID(), originalError, err)
					}
				} else {
					err := req.Pause().Send()
					if err != nil {
						a.logger.Errorf("failed to send pause '%s/%s' success: %v", req.EnvironmentID(), req.ExecutionID(), err)
					}
				}
			case cloud.RunnerRequestType_RESUME:
				originalError := a.runner.Resume(req.ExecutionID())
				if originalError != nil {
					err := req.SendError(originalError)
					if err != nil {
						a.logger.Errorf("failed to send resume '%s/%s' error: %v: %v", req.EnvironmentID(), req.ExecutionID(), originalError, err)
					}
				} else {
					err := req.Resume().Send()
					if err != nil {
						a.logger.Errorf("failed to send resume '%s/%s' success: %v", req.EnvironmentID(), req.ExecutionID(), err)
					}
				}
			default:
				err := req.SendError(errors.New("unrecognized runner operation"))
				if err != nil {
					a.logger.Errorf("failed to send runner error for execution '%s/%s' because of unknown operation: %v", req.EnvironmentID(), req.ExecutionID(), err)
				}
			}
		}(req)
	}
	wg.Wait()
	return watcher.Err()
}

func (a *agentLoop) runTestWorkflow(environmentId string, executionId string, executionToken string) error {
	_, err, _ := a.sf.Do(environmentId+"."+executionId, func() (interface{}, error) {
		return nil, a.directRunTestWorkflow(environmentId, executionId, executionToken)
	})

	return err
}

func (a *agentLoop) directRunTestWorkflow(environmentId string, executionId string, executionToken string) error {
	ctx := context.Background()
	logger := a.logger.With("environmentId", environmentId, "executionId", executionId)

	// Get the execution details
	execution, err := a.client.GetExecution(ctx, environmentId, executionId)
	if err != nil {
		return errors2.Wrapf(err, "failed to get execution details '%s/%s' from Control Plane", environmentId, executionId)
	}
	if execution.RunnerId != a.proContext.Agent.ID && execution.RunnerId != "" {
		return errors.New("execution is assigned to a different runner")
	}

	// Inform that everything is fine, because the execution is already there.
	if execution.Result != nil && !execution.Result.IsQueued() {
		return nil
	}

	parentIds := ""
	if execution.RunningContext != nil && execution.RunningContext.Actor != nil {
		parentIds = execution.RunningContext.Actor.ExecutionPath
	}
	result, err := a.runner.Execute(executionworkertypes.ExecuteRequest{
		Token: executionToken,
		Execution: testworkflowconfig.ExecutionConfig{
			Id:               execution.Id,
			GroupId:          execution.GroupId,
			Name:             execution.Name,
			Number:           execution.Number,
			ScheduledAt:      execution.ScheduledAt,
			DisableWebhooks:  execution.DisableWebhooks,
			Debug:            false,
			OrganizationId:   a.organizationId,
			OrganizationSlug: a.proContext.OrgSlug,
			EnvironmentId:    environmentId,
			EnvironmentSlug:  a.proContext.GetEnvSlug(environmentId),
			ParentIds:        parentIds,
		},
		Workflow:     testworkflowmappers.MapTestWorkflowAPIToKube(*execution.ResolvedWorkflow),
		ControlPlane: a.controlPlaneConfig, // TODO: fetch it from the control plane?
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

	// Inform that everything is fine, because the execution is already there.
	if result.Redundant {
		return nil
	}

	// Apply the known data to temporary object.
	execution.Namespace = result.Namespace
	execution.Signature = result.Signature
	execution.RunnerId = a.proContext.Agent.ID
	execution.AssignedAt = time.Now()
	if err = a.init(context.Background(), environmentId, execution); err != nil {
		logger.Errorw("failed to mark execution as initialized", "error", err)
	}

	return nil
}
