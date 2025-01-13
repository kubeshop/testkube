package runner

import (
	"context"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/log"
	configrepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/ui"
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
	grpcApiToken      string
	grpcClient        cloud.TestKubeCloudAPIClient
	proContext        config.ProContext
	worker            executionworkertypes.Worker
	runner            Runner
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
	grpcClient cloud.TestKubeCloudAPIClient,
	grpcApiToken string,
	proContext config.ProContext,
	executionWorker executionworkertypes.Worker,
	opts Options,
) Service {
	resultsRepository := cloudtestworkflow.NewCloudRepository(grpcClient, grpcApiToken)

	return &service{
		runnerId:          runnerId,
		logger:            logger,
		eventsEmitter:     eventsEmitter,
		grpcApiToken:      grpcApiToken,
		grpcClient:        grpcClient,
		proContext:        proContext,
		resultsRepository: resultsRepository,
		worker:            executionWorker,
		opts:              opts,
		runner: New(
			runnerId,
			executionWorker,
			cloudtestworkflow.NewCloudOutputRepository(grpcClient, grpcApiToken, opts.StorageSkipVerify),
			resultsRepository,
			configClient,
			grpcClient,
			grpcApiToken,
			eventsEmitter,
			metricsClient,
			proContext,
			opts.DashboardURI,
			opts.StorageSkipVerify,
			opts.NewExecutionsEnabled,
		),
	}
}

func (s *service) recover(ctx context.Context) (err error) {
	if !s.opts.NewExecutionsEnabled {
		fmt.Println(ui.Green("recover legacy"))
		return s.recoverLegacy(ctx)
	}

	md := metadata.New(map[string]string{apiKeyMeta: s.grpcApiToken, orgIdMeta: s.proContext.OrgID, agentIdMeta: s.runnerId})
	opts := []grpc.CallOption{grpc.UseCompressor(gzip.Name), grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	executions, err := s.grpcClient.GetUnfinishedExecutions(metadata.NewOutgoingContext(ctx, md), &emptypb.Empty{}, opts...)
	for {
		// Take the context error if possible
		if err == nil && ctx.Err() != nil {
			err = ctx.Err()
		}

		// End when it's done
		if errors.Is(err, io.EOF) {
			return nil
		}

		// Handle the error
		if err != nil {
			log.DefaultLogger.Errorw("failed to get runner executions", "error", err)
			return err
		}

		// Get the next execution to monitor
		var exec *cloud.UnfinishedExecution
		exec, err = executions.Recv()
		if err != nil {
			continue
		}

		// TODO: Pass hints (namespace, signature, scheduledAt)
		go func(environmentId string, executionId string) {
			err := s.runner.Monitor(ctx, s.proContext.OrgID, environmentId, executionId)
			if err != nil {
				s.logger.Errorw("failed to monitor execution", "id", executionId, "error", err)
			}
		}(exec.EnvironmentId, exec.Id)
	}
}

func (s *service) recoverLegacy(ctx context.Context) (err error) {
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
		s.grpcClient,
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
