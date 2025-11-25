//nolint:staticcheck
package controlplane

import (
	"context"
	"math"
	"net"
	"time"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudexecutor "github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
	"github.com/kubeshop/testkube/pkg/repository"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	domainstorage "github.com/kubeshop/testkube/pkg/storage"
)

const (
	HealthCheckInterval = 60 * time.Second
	SendPingInterval    = HealthCheckInterval / 2
)

type Server struct {
	cloud.UnimplementedTestKubeCloudAPIServer
	executionv1.UnimplementedTestWorkflowExecutionServiceServer
	cfg       Config
	server    *grpc.Server
	commands  map[cloudexecutor.Command]CommandHandler
	enqueuer  scheduling.Enqueuer
	scheduler scheduling.Scheduler

	// note: encapsulation is broken here because the HTTP Server cannot yet be pulled into the build-in control plane
	//       until the commercial control plane becomes its own source of truth.
	ExecutionController         scheduling.Controller
	executionQuerier            scheduling.ExecutionQuerier
	storageClient               domainstorage.Client
	testWorkflowsClient         testworkflowclient.TestWorkflowClient
	testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient
	resultsRepository           testworkflow.Repository
	outputRepository            testworkflow.OutputRepository
	repositoryManager           repository.DatabaseRepository
	emitter                     *event.Emitter
	envID                       string // Environment ID for event grouping
}

type Config struct {
	Port                             int
	Verbose                          bool
	Logger                           *zap.SugaredLogger
	StorageBucket                    string
	FeatureTestWorkflowsCloudStorage bool
}

func New(
	cfg Config,
	enqueuer scheduling.Enqueuer,
	scheduler scheduling.Scheduler,
	executionController scheduling.Controller,
	executionQuerier scheduling.ExecutionQuerier,
	eventEmitter *event.Emitter,
	storageClient domainstorage.Client,
	testWorkflowsClient testworkflowclient.TestWorkflowClient,
	testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient,
	resultsRepository testworkflow.Repository,
	outputRepository testworkflow.OutputRepository,
	repositoryManager repository.DatabaseRepository,
	envID string,
	commandGroups ...CommandHandlers,
) *Server {
	commands := make(map[cloudexecutor.Command]CommandHandler)
	for _, group := range commandGroups {
		for cmd, handler := range group {
			commands[cmd] = handler
		}
	}
	return &Server{
		cfg:                         cfg,
		enqueuer:                    enqueuer,
		scheduler:                   scheduler,
		ExecutionController:         executionController,
		executionQuerier:            executionQuerier,
		commands:                    commands,
		storageClient:               storageClient,
		testWorkflowsClient:         testWorkflowsClient,
		testWorkflowTemplatesClient: testWorkflowTemplatesClient,
		resultsRepository:           resultsRepository,
		outputRepository:            outputRepository,
		repositoryManager:           repositoryManager,
		emitter:                     eventEmitter,
		envID:                       envID,
	}
}

func (s *Server) GetRepositoryManager() repository.DatabaseRepository {
	return s.repositoryManager
}

func (s *Server) Start(ctx context.Context, ln net.Listener) error {
	var opts []grpc.ServerOption

	// Create a server, make sure we put the grpcctxtags context before everything else.
	creds := insecure.NewCredentials()

	// default MaxRecvMsgSize is 4Mib, which causes trouble
	opts = append(opts,
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(math.MaxInt32),
		grpc.ChainUnaryInterceptor(grpcrecovery.UnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(grpcrecovery.StreamServerInterceptor()),
	)
	if s.cfg.Verbose {
		// Shared options for the logger, with a custom gRPC code to log level function.
		logger := s.cfg.Logger.Desugar()
		grpczap.ReplaceGrpcLoggerV2(logger)
		opts = append(
			opts,
			grpc.ChainUnaryInterceptor(
				grpcctxtags.UnaryServerInterceptor(grpcctxtags.WithFieldExtractor(grpcctxtags.CodeGenRequestFieldExtractor)),
				grpczap.UnaryServerInterceptor(logger),
			),
		)
		opts = append(
			opts,
			grpc.ChainStreamInterceptor(
				grpcctxtags.StreamServerInterceptor(grpcctxtags.WithFieldExtractor(grpcctxtags.CodeGenRequestFieldExtractor)),
				grpczap.StreamServerInterceptor(logger),
			),
		)
	}
	opts = append(opts,
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{PermitWithoutStream: true}),
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: client.GRPCKeepaliveTime, Timeout: client.GRPCKeepaliveTimeout}))
	grpcServer := grpc.NewServer(opts...)

	cloud.RegisterTestKubeCloudAPIServer(grpcServer, s)
	executionv1.RegisterTestWorkflowExecutionServiceServer(grpcServer, s)
	s.server = grpcServer
	go func() {
		<-ctx.Done()
		s.Shutdown()
	}()
	err := grpcServer.Serve(ln)
	if err != nil {
		return errors.Wrap(err, "grpc server error")
	}
	return nil
}

func (s *Server) Shutdown() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}
