package runner

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/log"
	configrepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

type Options struct {
	ClusterID           string
	DefaultNamespace    string
	ServiceAccountNames map[string]string

	StorageSkipVerify bool

	GlobalTemplate GlobalTemplateFactory
}

type service struct {
	logger             *zap.SugaredLogger
	eventsEmitter      event.Interface
	client             controlplaneclient.Client
	controlPlaneConfig testworkflowconfig.ControlPlaneConfig
	proContext         config.ProContext
	worker             executionworkertypes.Worker
	runner             Runner
	opts               Options
}

type Service interface {
	Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error)
	Start(ctx context.Context) error
}

func NewService(
	logger *zap.SugaredLogger,
	eventsEmitter event.Interface,
	metricsClient metrics.Metrics,
	configClient configrepo.Repository,
	client controlplaneclient.Client,
	controlPlaneConfig testworkflowconfig.ControlPlaneConfig,
	proContext config.ProContext,
	executionWorker executionworkertypes.Worker,
	opts Options,
) Service {
	return &service{
		logger:             logger,
		eventsEmitter:      eventsEmitter,
		client:             client,
		controlPlaneConfig: controlPlaneConfig,
		proContext:         proContext,
		worker:             executionWorker,
		opts:               opts,
		runner: New(
			executionWorker,
			configClient,
			client,
			eventsEmitter,
			metricsClient,
			proContext,
			opts.StorageSkipVerify,
			opts.GlobalTemplate,
		),
	}
}

func (s *service) reattach(ctx context.Context) (err error) {
	executions, err := s.client.GetRunnerOngoingExecutions(ctx)
	if err != nil {
		log.DefaultLogger.Errorw("failed to get runner executions", "error", err)
		return
	}

	for _, exec := range executions {
		go func(environmentId string, executionId string) {
			err := s.runner.Monitor(context.Background(), s.proContext.OrgID, environmentId, executionId)
			if err == nil {
				s.logger.Infow("Reattached execution", "executionId", executionId)
				return
			}
			if !errors.Is(err, registry.ErrResourceNotFound) && !errors.Is(err, controller.ErrJobAborted) {
				s.logger.Errorw("failed to monitor execution", "id", executionId, "error", err)
				return
			}

			s.logger.Warnw("execution to monitor not found. reattaching again.", "id", executionId, "error", err)

			// Get the existing execution
			execution, err := s.client.GetExecution(ctx, environmentId, executionId)
			if err != nil {
				s.logger.Errorw("failed to reattach to execution: getting execution", "id", executionId, "error", err)
				return
			}

			// Ignore if it's still queued - orchestrator will reattach to it later
			if execution.Result.IsQueued() {
				s.logger.Warnw("execution to monitor is still queued: leaving it for orchestrator", "id", executionId)
				return
			}

			// Check if there is error message acknowledged
			sigSequence := stage.MapSignatureListToInternal(stage.MapSignatureToSequence(stage.MapSignatureList(execution.Signature)))
			errorMessage := execution.Result.Initialization.ErrorMessage
			if errorMessage == "" {
				for _, sig := range sigSequence {
					if execution.Result.Steps[sig.Ref].ErrorMessage != "" {
						errorMessage = execution.Result.Steps[sig.Ref].ErrorMessage
						break
					}
				}
			}

			// Finalize and save the result
			execution.Result.HealAbortedOrCanceled(sigSequence, errorMessage, controller.DefaultErrorMessage, "aborted")
			execution.Result.HealTimestamps(sigSequence, execution.ScheduledAt, time.Time{}, time.Time{}, true)
			execution.Result.HealDuration(execution.ScheduledAt)
			execution.Result.HealMissingPauseStatuses()
			execution.Result.HealStatus(sigSequence)
			if err = s.client.FinishExecutionResult(ctx, environmentId, executionId, execution.Result); err != nil {
				s.logger.Errorw("failed to recover execution: saving execution", "id", executionId, "error", err)
			} else {
				s.logger.Infow("recovered execution", "id", executionId, "error", err)
			}
		}(exec.EnvironmentId, exec.Id)
	}

	return
}

func (s *service) start(ctx context.Context) (err error) {
	return newAgentLoop(
		s.runner,
		s.worker,
		s.logger,
		s.eventsEmitter,
		s.client,
		s.controlPlaneConfig, // TODO: fetch it from the control plane?
		s.proContext,
		s.proContext.OrgID,
		s.proContext.EnvID,
	).Start(ctx)
}

func (s *service) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.reattach(ctx)
	})

	g.Go(func() error {
		return s.start(ctx)
	})

	return g.Wait()
}

func (s *service) Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	return s.runner.Execute(request)
}
