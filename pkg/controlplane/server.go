//nolint:staticcheck
package controlplane

import (
	"context"
	"math"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
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
	"github.com/kubeshop/testkube/pkg/grpcutils"
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
	startEvents                 *startEventDispatcher
	envID                       string // Environment ID for event grouping
}

type Config struct {
	Port                             int
	Verbose                          bool
	Logger                           *zap.SugaredLogger
	StorageBucket                    string
	FeatureTestWorkflowsCloudStorage bool

	// DispatchBatchSize bounds how many executions one GetExecutionUpdates poll
	// hands to the runner.
	DispatchBatchSize int
	// RedispatchAfter is how long a dispatched execution may sit in STARTING
	// before it is offered again.
	RedispatchAfter time.Duration
	// StartTimeout is how long a dispatched execution may sit in STARTING before
	// the reaper fails it explicitly.
	StartTimeout time.Duration
	// ReaperInterval is how often the reaper runs.
	ReaperInterval time.Duration
}

// Defaults for the dispatch knobs.
//
// They are applied by the accessors below rather than in New, because tests
// construct &Server{} directly and a zero Config has to behave.
const (
	DefaultDispatchBatchSize = 25
	DefaultRedispatchAfter   = time.Minute
	DefaultStartTimeout      = 10 * time.Minute
	DefaultReaperInterval    = time.Minute
)

func (s *Server) dispatchBatchSize() int {
	if s.cfg.DispatchBatchSize <= 0 {
		return DefaultDispatchBatchSize
	}
	return s.cfg.DispatchBatchSize
}

func (s *Server) redispatchAfter() time.Duration {
	if s.cfg.RedispatchAfter <= 0 {
		return DefaultRedispatchAfter
	}
	return s.cfg.RedispatchAfter
}

// startTimeout is how long a row may stay in STARTING before it is failed.
//
// Clamped to stay above redispatchAfter: if it were the shorter of the two the
// reaper would fail executions that had not yet been offered a second time.
func (s *Server) startTimeout() time.Duration {
	timeout := s.cfg.StartTimeout
	if timeout <= 0 {
		timeout = DefaultStartTimeout
	}
	if redispatch := s.redispatchAfter(); timeout <= redispatch {
		return redispatch * 2
	}
	return timeout
}

func (s *Server) reaperInterval() time.Duration {
	if s.cfg.ReaperInterval <= 0 {
		return DefaultReaperInterval
	}
	return s.cfg.ReaperInterval
}

func New(
	cfg Config,
	enqueuer scheduling.Enqueuer,
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
	srv := &Server{
		cfg:                         cfg,
		enqueuer:                    enqueuer,
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
	srv.startEvents = newStartEventDispatcher(
		resultsRepository,
		eventEmitter,
		envID,
		cfg.Logger,
		srv.dispatchBatchSize()*startEventBufferBatches,
	)
	return srv
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
		opts = append(
			opts,
			grpc.ChainUnaryInterceptor(
				logging.UnaryServerInterceptor(grpcutils.ZapGRPCLogger(logger)),
			),
		)
		opts = append(
			opts,
			grpc.ChainStreamInterceptor(
				logging.StreamServerInterceptor(grpcutils.ZapGRPCLogger(logger)),
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

	// Publishing start events and reaping stale dispatches both run off the
	// request path, so that neither a slow webhook subscriber nor a backlog of
	// unacknowledged executions can delay the poll the runner is blocked on.
	if s.startEvents != nil {
		s.startEvents.run(ctx)
		// Queued events would otherwise be lost on a restart, and nothing records
		// that a start event was published, so there is no way to notice.
		defer s.startEvents.drain()
	}
	go s.reapStaleDispatches(ctx)

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
