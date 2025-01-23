package runner

import (
	"context"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/log"
	configrepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
)

type Options struct {
	ClusterID           string
	DashboardURI        string
	DefaultNamespace    string
	ServiceAccountNames map[string]string

	StorageSkipVerify bool

	ControlPlaneStorageEnabled bool
	NewArchitectureEnabled     bool
}

type service struct {
	runnerId           string
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
	runnerId string,
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
		runnerId:           runnerId,
		logger:             logger,
		eventsEmitter:      eventsEmitter,
		client:             client,
		controlPlaneConfig: controlPlaneConfig,
		proContext:         proContext,
		worker:             executionWorker,
		opts:               opts,
		runner: New(
			runnerId,
			executionWorker,
			configClient,
			client,
			eventsEmitter,
			metricsClient,
			proContext,
			opts.DashboardURI,
			opts.StorageSkipVerify,
			opts.NewArchitectureEnabled,
		),
	}
}

func (s *service) recover(ctx context.Context) (err error) {
	executions, err := s.client.GetRunnerOngoingExecutions(ctx)
	if err != nil {
		log.DefaultLogger.Errorw("failed to get runner executions", "error", err)
		return
	}

	for _, exec := range executions {
		go func(environmentId string, executionId string) {
			err := s.runner.Monitor(ctx, s.proContext.OrgID, environmentId, executionId)
			if err != nil {
				s.logger.Errorw("failed to monitor execution", "id", executionId, "error", err)
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
		s.runnerId,
		s.proContext.OrgID,
		s.proContext.EnvID,
		s.opts.NewArchitectureEnabled,
	).Start(ctx)
}

func (s *service) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.recover(ctx)
	})

	g.Go(func() error {
		return s.start(ctx)
	})

	return g.Wait()
}

func (s *service) Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	return s.runner.Execute(request)
}
