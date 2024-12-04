package controlplane

import (
	"context"
	"encoding/json"
	errors2 "errors"
	"fmt"
	"math"
	"net"
	"os"
	"time"

	"github.com/gofiber/fiber/v2/log"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	log2 "github.com/kubeshop/testkube/pkg/log"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/runner"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

const (
	KeepAliveTime       = 10 * time.Second
	KeepAliveTimeout    = 5 * time.Second
	HealthCheckInterval = 60 * time.Second
	SendPingInterval    = HealthCheckInterval / 2
)

type EventEmitter interface {
	Notify(event testkube.Event)
}

type Server struct {
	cloud.UnimplementedTestKubeCloudAPIServer
	cfg             Config
	server          *grpc.Server
	getEventEmitter func() EventEmitter
	getRunner       func() runner.Runner
	secretManager   secretmanager.SecretManager
	scheduler       *scheduler
	dashboardUri    string
	commands        map[executor.Command]CommandHandler
}

type Config struct {
	Port    int
	Verbose bool
	Logger  *zap.SugaredLogger
}

func New(
	cfg Config,
	executionScheduler *testworkflowexecutor.ExecutionScheduler,
	dashboardUri string,
	commandGroups ...CommandHandlers,
) *Server {
	commands := make(map[executor.Command]CommandHandler)
	for _, group := range commandGroups {
		for cmd, handler := range group {
			commands[cmd] = handler
		}
	}
	return &Server{
		cfg:             cfg,
		dashboardUri:    dashboardUri,
		secretManager:   executionScheduler.SecretManager(),
		getEventEmitter: func() EventEmitter { return executionScheduler.Emitter() },
		getRunner:       executionScheduler.Runner,
		scheduler: NewScheduler(
			executionScheduler.TestWorkflowsClient(),
			executionScheduler.TestWorkflowsTemplatesClient(),
			executionScheduler.Repository(),
			executionScheduler.OutputRepository(),
			executionScheduler.GlobalTemplateName(),
			"",
		),
		commands: commands,
	}
}

func (s *Server) GetProContext(_ context.Context, _ *emptypb.Empty) (*cloud.ProContextResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not supported in the standalone version")
}

func (s *Server) GetCredential(_ context.Context, _ *cloud.CredentialRequest) (*cloud.CredentialResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not supported in the standalone version")
}

func (s *Server) ExecuteAsync(srv cloud.TestKubeCloudAPI_ExecuteAsyncServer) error {
	ctx, cancel := context.WithCancel(srv.Context())
	g, _ := errgroup.WithContext(ctx)
	defer cancel()

	// Ignore all the messages
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				srv.Recv()
			}
		}
	})

	g.Go(func() error {
		ticker := time.NewTicker(HealthCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				messageId := fmt.Sprintf("hs%d", time.Now().UnixNano())
				req := &cloud.ExecuteRequest{Url: "healthcheck", MessageId: messageId}
				err := srv.Send(req)
				if err != nil {
					log.Errorw("failed to publish agent healthcheck", "error", err)
				}
			}
		}
	})

	return g.Wait()
}

// TODO: Consider deleting that
func (s *Server) GetTestWorkflowNotificationsStream(srv cloud.TestKubeCloudAPI_GetTestWorkflowNotificationsStreamServer) error {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()
	g, _ := errgroup.WithContext(ctx)

	ticker := time.NewTicker(SendPingInterval)
	defer ticker.Stop()

	g.Go(func() error {
		for {
			select {
			case <-ticker.C:
				srv.Send(&cloud.TestWorkflowNotificationsRequest{
					StreamId:    "ping",
					ExecutionId: "ping",
					RequestType: cloud.TestWorkflowNotificationsRequestType_WORKFLOW_STREAM_HEALTH_CHECK,
				})
			case <-ctx.Done():
				return nil
			}
		}
	})

	// Ignore all the messages
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				srv.Recv()
			}
		}
	})

	return g.Wait()
}

// Send is called on agent client, returning from this method closes the connection
func (s *Server) Send(srv cloud.TestKubeCloudAPI_SendServer) error {
	for {
		if err := srv.Context().Err(); err != nil {
			log.Info("agent websocket stream is canceled, agent client is disconnected")
			return nil
		}

		_, err := srv.Recv()
		if err != nil {
			errMsg := "failed to receive websocket message"
			log.Errorw(errMsg, "error", err)
			return errors.Wrap(err, errMsg)
		}
	}
}

// TODO: Consider deleting that
func (s *Server) GetLogsStream(srv cloud.TestKubeCloudAPI_GetLogsStreamServer) error {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()
	g, _ := errgroup.WithContext(ctx)

	ticker := time.NewTicker(SendPingInterval)
	defer ticker.Stop()

	g.Go(func() error {
		for {
			select {
			case <-ticker.C:
				srv.Send(&cloud.LogsStreamRequest{
					StreamId:    "ping",
					ExecutionId: "ping",
					RequestType: cloud.LogsStreamRequestType_STREAM_HEALTH_CHECK,
				})
			case <-ctx.Done():
				return nil
			}
		}
	})

	// Ignore all the messages
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				srv.Recv()
			}
		}
	})

	return g.Wait()
}

func (s *Server) Call(ctx context.Context, request *cloud.CommandRequest) (*cloud.CommandResponse, error) {
	if cmd, ok := s.commands[executor.Command(request.Command)]; ok {
		return cmd(ctx, request)
	}
	return nil, errors.Errorf("command not implemented: %s", request.Command)
}

func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return errors.Errorf("failed to listen for GraphQL server: %v", err)
	}
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
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: KeepAliveTime, Timeout: KeepAliveTimeout}))
	grpcServer := grpc.NewServer(opts...)

	cloud.RegisterTestKubeCloudAPIServer(grpcServer, s)
	s.server = grpcServer
	go func() {
		<-ctx.Done()
		s.Shutdown()
	}()
	err = grpcServer.Serve(ln)
	if err != nil {
		return errors.Wrap(err, "grpc server error")
	}
	return nil
}

func retry(count int, delayBase time.Duration, fn func() error) (err error) {
	for i := 0; i < count; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(i) * delayBase)
	}
	return err
}

func (s *Server) ScheduleExecution(req *cloud.ScheduleRequest, srv cloud.TestKubeCloudAPI_ScheduleExecutionServer) error {
	// Prepare dependencies
	eventEmitter := s.getEventEmitter()
	runner := s.getRunner()
	sensitiveDataHandler := NewSecretHandler(s.secretManager)

	// Schedule execution
	ch, err := s.scheduler.Schedule(srv.Context(), sensitiveDataHandler, req)
	if err != nil {
		return err
	}

	controlPlaneConfig := testworkflowconfig.ControlPlaneConfig{
		DashboardUrl:   s.dashboardUri,
		CDEventsTarget: os.Getenv("CDEVENTS_TARGET"),
	}

	for execution := range ch {
		eventEmitter.Notify(testkube.NewEventQueueTestWorkflow(execution))

		// Send the data
		v, err := json.Marshal(execution)
		if err != nil {
			return err
		}
		if err = srv.Send(&cloud.ScheduleResponse{Execution: v}); err != nil {
			// TODO: retry?
		}

		// Finish early if it's immediately known to finish
		if execution.Result.IsFinished() {
			eventEmitter.Notify(testkube.NewEventStartTestWorkflow(execution))
			if execution.Result.IsAborted() {
				eventEmitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
			} else if execution.Result.IsFailed() {
				eventEmitter.Notify(testkube.NewEventEndTestWorkflowFailed(execution))
			} else {
				eventEmitter.Notify(testkube.NewEventEndTestWorkflowSuccess(execution))
			}
			continue
		}

		// Start the execution
		// TODO: Make it another gRPC operation
		parentIds := ""
		if execution.RunningContext != nil && execution.RunningContext.Actor != nil {
			parentIds = execution.RunningContext.Actor.ExecutionPath
		}
		result, err := runner.Execute(executionworkertypes.ExecuteRequest{
			Execution: testworkflowconfig.ExecutionConfig{
				Id:              execution.Id,
				GroupId:         execution.GroupId,
				Name:            execution.Name,
				Number:          execution.Number,
				ScheduledAt:     execution.ScheduledAt,
				DisableWebhooks: execution.DisableWebhooks,
				Debug:           false,
				OrganizationId:  "",
				EnvironmentId:   "",
				ParentIds:       parentIds,
			},
			Secrets:      sensitiveDataHandler.Get(execution.Id),
			Workflow:     testworkflowmappers.MapTestWorkflowAPIToKube(*execution.ResolvedWorkflow),
			ControlPlane: controlPlaneConfig,
		})

		// TODO: define "revoke" error by runner (?)
		if err != nil {
			err2 := s.scheduler.CriticalError(execution, "Failed to run execution", err)
			err = errors2.Join(err, err2)
			if err != nil {
				log2.DefaultLogger.Errorw("failed to run and update execution", "executionId", execution.Id, "error", err)
			}
			eventEmitter.Notify(testkube.NewEventStartTestWorkflow(execution))
			eventEmitter.Notify(testkube.NewEventEndTestWorkflowAborted(execution))
			continue
		}

		// Inform about execution start
		eventEmitter.Notify(testkube.NewEventStartTestWorkflow(execution))

		// Apply the known data to temporary object.
		execution.Namespace = result.Namespace
		execution.Signature = result.Signature
		execution.RunnerId = ""
		if err = s.scheduler.Start(execution); err != nil {
			log2.DefaultLogger.Errorw("failed to mark execution as initialized", "executionId", execution.Id, "error", err)
		}
	}

	return nil
}

// TODO: Use this when context is down
func (s *Server) Shutdown() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}
