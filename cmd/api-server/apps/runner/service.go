package runner

import (
	"context"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/event"
	configrepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/runner"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

type Options struct {
	ClusterID           string
	DashboardURI        string
	DefaultNamespace    string
	ServiceAccountNames map[string]string

	StorageSkipVerify bool

	ControlPlaneStorageEnabled bool
	NewExecutionsEnabled       bool
}

type service struct {
	runnerId          string
	logger            *zap.SugaredLogger
	eventsEmitter     event.Interface
	configClient      configrepo.Repository
	grpcConn          *grpc.ClientConn
	grpcApiToken      string
	grpcClient        cloud.TestKubeCloudAPIClient
	proContext        config.ProContext
	worker            executionworkertypes.Worker
	runner            runner.Runner
	resultsRepository testworkflow.Repository
	opts              Options
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
	grpcConn *grpc.ClientConn,
	grpcApiToken string,
	proContext config.ProContext,
	executionWorker executionworkertypes.Worker,
	opts Options,
) Service {
	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)
	resultsRepository := cloudtestworkflow.NewCloudRepository(grpcClient, grpcConn, grpcApiToken)

	return &service{
		runnerId:          runnerId,
		logger:            logger,
		eventsEmitter:     eventsEmitter,
		grpcConn:          grpcConn,
		grpcApiToken:      grpcApiToken,
		grpcClient:        grpcClient,
		proContext:        proContext,
		resultsRepository: resultsRepository,
		worker:            executionWorker,
		opts:              opts,
		runner: runner.New(
			executionWorker,
			cloudtestworkflow.NewCloudOutputRepository(grpcClient, grpcConn, grpcApiToken, opts.StorageSkipVerify),
			resultsRepository,
			configClient,
			grpcConn,
			grpcApiToken,
			eventsEmitter,
			metricsClient,
			opts.DashboardURI,
			opts.StorageSkipVerify,
			opts.NewExecutionsEnabled,
		),
	}
}

func (s *service) recover(ctx context.Context) (err error) {
	var list []testkube.TestWorkflowExecution
	for {
		// TODO: it should get running only in the context of current runner (worker.List?)
		list, err = s.resultsRepository.GetRunning(ctx)
		if err != nil {
			s.logger.Errorw("failed to fetch running executions to recover", "error", err)
			<-time.After(time.Second)
			continue
		}
		break
	}

	for i := range list {
		if (list[i].RunnerId == "" && len(list[i].Signature) == 0) || (list[i].RunnerId != "" && list[i].RunnerId != s.runnerId) {
			continue
		}

		// TODO: Should it throw error at all?
		// TODO: Pass hints (namespace, signature, scheduledAt)
		go func(e *testkube.TestWorkflowExecution) {
			err := s.runner.Monitor(ctx, s.proContext.OrgID, s.proContext.EnvID, e.Id)
			if err != nil {
				s.logger.Errorw("failed to monitor execution", "id", e.Id, "error", err)
			}
		}(&list[i])
	}

	return
}

func (s *service) start(ctx context.Context) (err error) {
	return newAgentLoop(
		s.runner,
		s.worker,
		s.logger,
		s.eventsEmitter,
		s.grpcConn,
		s.grpcApiToken,
		s.runnerId,
		s.proContext.OrgID,
		s.proContext.EnvID,
		s.opts.NewExecutionsEnabled,
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
