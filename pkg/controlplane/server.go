package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
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

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudexecutor "github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/repository"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	domainstorage "github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

const (
	KeepAliveTime       = 10 * time.Second
	KeepAliveTimeout    = 5 * time.Second
	HealthCheckInterval = 60 * time.Second
	SendPingInterval    = HealthCheckInterval / 2
)

type Server struct {
	cloud.UnimplementedTestKubeCloudAPIServer
	cfg                         Config
	server                      *grpc.Server
	commands                    map[cloudexecutor.Command]CommandHandler
	executor                    testworkflowexecutor.TestWorkflowExecutor
	storageClient               domainstorage.Client
	testWorkflowsClient         testworkflowclient.TestWorkflowClient
	testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient
	resultsRepository           testworkflow.Repository
	outputRepository            testworkflow.OutputRepository
	repositoryManager           repository.DatabaseRepository
}

type Config struct {
	Port                             int
	Verbose                          bool
	Logger                           *zap.SugaredLogger
	StorageBucket                    string
	FeatureNewArchitecture           bool
	FeatureTestWorkflowsCloudStorage bool
}

// TODO: Check if runner works fine
func New(
	cfg Config,
	executor testworkflowexecutor.TestWorkflowExecutor,
	storageClient domainstorage.Client,
	testWorkflowsClient testworkflowclient.TestWorkflowClient,
	testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient,
	resultsRepository testworkflow.Repository,
	outputRepository testworkflow.OutputRepository,
	repositoryManager repository.DatabaseRepository,
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
		executor:                    executor,
		commands:                    commands,
		storageClient:               storageClient,
		testWorkflowsClient:         testWorkflowsClient,
		testWorkflowTemplatesClient: testWorkflowTemplatesClient,
		resultsRepository:           resultsRepository,
		outputRepository:            outputRepository,
		repositoryManager:           repositoryManager,
	}
}

func (s *Server) GetProContext(_ context.Context, _ *emptypb.Empty) (*cloud.ProContextResponse, error) {
	caps := make([]*cloud.Capability, 0)
	if s.cfg.FeatureNewArchitecture {
		caps = append(caps, &cloud.Capability{Name: string(capabilities.CapabilityNewArchitecture), Enabled: true})
	}
	if s.cfg.FeatureTestWorkflowsCloudStorage {
		caps = append(caps, &cloud.Capability{Name: string(capabilities.CapabilityCloudStorage), Enabled: true})
	}
	return &cloud.ProContextResponse{Capabilities: caps}, nil
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

func (s *Server) GetEventStream(_ *cloud.EventStreamRequest, srv cloud.TestKubeCloudAPI_GetEventStreamServer) error {
	// Do nothing - it doesn't need to pass events down
	<-srv.Context().Done()
	return nil
}

func (s *Server) GetRunnerRequests(srv cloud.TestKubeCloudAPI_GetRunnerRequestsServer) error {
	// Do nothing - it doesn't need to send runner requests
	<-srv.Context().Done()
	return nil
}

func (s *Server) InitExecution(ctx context.Context, req *cloud.InitExecutionRequest) (*cloud.InitExecutionResponse, error) {
	var signature []testkube.TestWorkflowSignature
	err := json.Unmarshal(req.Signature, &signature)
	if err != nil {
		return nil, err
	}
	err = s.resultsRepository.Init(ctx, req.Id, testworkflow.InitData{RunnerID: "oss", Namespace: req.Namespace, Signature: signature, AssignedAt: time.Now()})
	if err != nil {
		return nil, err
	}
	return &cloud.InitExecutionResponse{}, nil
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

func (s *Server) GetTestWorkflowServiceNotificationsStream(srv cloud.TestKubeCloudAPI_GetTestWorkflowServiceNotificationsStreamServer) error {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()
	g, _ := errgroup.WithContext(ctx)

	ticker := time.NewTicker(SendPingInterval)
	defer ticker.Stop()

	g.Go(func() error {
		for {
			select {
			case <-ticker.C:
				srv.Send(&cloud.TestWorkflowServiceNotificationsRequest{
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

func (s *Server) GetTestWorkflowParallelStepNotificationsStream(srv cloud.TestKubeCloudAPI_GetTestWorkflowParallelStepNotificationsStreamServer) error {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()
	g, _ := errgroup.WithContext(ctx)

	ticker := time.NewTicker(SendPingInterval)
	defer ticker.Stop()

	g.Go(func() error {
		for {
			select {
			case <-ticker.C:
				srv.Send(&cloud.TestWorkflowParallelStepNotificationsRequest{
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
	if cmd, ok := s.commands[cloudexecutor.Command(request.Command)]; ok {
		return cmd(ctx, request)
	}
	return nil, errors.Errorf("command not implemented: %s", request.Command)
}

func (s *Server) Start(ctx context.Context) error {
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

func (s *Server) ScheduleExecution(req *cloud.ScheduleRequest, srv cloud.TestKubeCloudAPI_ScheduleExecutionServer) error {
	resp := s.executor.Execute(srv.Context(), "", req)
	for execution := range resp.Channel() {
		// Send the data
		// TODO: Use protobuf struct?
		v, err := json.Marshal(execution)
		if err != nil {
			return err
		}
		if err = srv.Send(&cloud.ScheduleResponse{Execution: v}); err != nil {
			// TODO: retry?
			return err
		}
	}
	if resp.Error() != nil {
		return resp.Error()
	}
	return nil
}

// TODO: Use this when context is down
func (s *Server) Shutdown() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}

func (s *Server) GetTestWorkflow(ctx context.Context, req *cloud.GetTestWorkflowRequest) (*cloud.GetTestWorkflowResponse, error) {
	workflow, err := s.testWorkflowsClient.Get(ctx, "", req.Name)
	if err != nil {
		return nil, err
	}
	workflowBytes, err := json.Marshal(workflow)
	if err != nil {
		return nil, err
	}
	return &cloud.GetTestWorkflowResponse{Workflow: workflowBytes}, nil
}

func (s *Server) ListTestWorkflows(req *cloud.ListTestWorkflowsRequest, srv cloud.TestKubeCloudAPI_ListTestWorkflowsServer) error {
	workflows, err := s.testWorkflowsClient.List(srv.Context(), "", testworkflowclient.ListOptions{
		Labels:     req.Labels,
		TextSearch: req.TextSearch,
		Offset:     req.Offset,
		Limit:      req.Limit,
	})
	if err != nil {
		return err
	}
	var workflowBytes []byte
	for _, workflow := range workflows {
		workflowBytes, err = json.Marshal(workflow)
		if err != nil {
			return err
		}
		err = srv.Send(&cloud.TestWorkflowListItem{Workflow: workflowBytes})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) ListTestWorkflowLabels(ctx context.Context, req *cloud.ListTestWorkflowLabelsRequest) (*cloud.ListTestWorkflowLabelsResponse, error) {
	labels, err := s.testWorkflowsClient.ListLabels(ctx, "")
	if err != nil {
		return nil, err
	}
	res := &cloud.ListTestWorkflowLabelsResponse{Labels: make([]*cloud.LabelListItem, 0, len(labels))}
	for k, v := range labels {
		res.Labels = append(res.Labels, &cloud.LabelListItem{Name: k, Value: v})
	}
	return res, nil
}

func (s *Server) CreateTestWorkflow(ctx context.Context, req *cloud.CreateTestWorkflowRequest) (*cloud.CreateTestWorkflowResponse, error) {
	var workflow testkube.TestWorkflow
	err := json.Unmarshal(req.Workflow, &workflow)
	if err != nil {
		return nil, err
	}
	err = s.testWorkflowsClient.Create(ctx, "", workflow)
	if err != nil {
		return nil, err
	}
	return &cloud.CreateTestWorkflowResponse{}, nil
}

func (s *Server) UpdateTestWorkflow(ctx context.Context, req *cloud.UpdateTestWorkflowRequest) (*cloud.UpdateTestWorkflowResponse, error) {
	var workflow testkube.TestWorkflow
	err := json.Unmarshal(req.Workflow, &workflow)
	if err != nil {
		return nil, err
	}
	err = s.testWorkflowsClient.Update(ctx, "", workflow)
	if err != nil {
		return nil, err
	}
	return &cloud.UpdateTestWorkflowResponse{}, nil
}

func (s *Server) DeleteTestWorkflow(ctx context.Context, req *cloud.DeleteTestWorkflowRequest) (*cloud.DeleteTestWorkflowResponse, error) {
	err := s.testWorkflowsClient.Delete(ctx, "", req.Name)
	if err != nil {
		return nil, err
	}
	return &cloud.DeleteTestWorkflowResponse{}, nil
}

func (s *Server) DeleteTestWorkflowsByLabels(ctx context.Context, req *cloud.DeleteTestWorkflowsByLabelsRequest) (*cloud.DeleteTestWorkflowsByLabelsResponse, error) {
	count, err := s.testWorkflowsClient.DeleteByLabels(ctx, "", req.Labels)
	if err != nil {
		return nil, err
	}
	return &cloud.DeleteTestWorkflowsByLabelsResponse{Count: count}, nil
}

func (s *Server) GetTestWorkflowTemplate(ctx context.Context, req *cloud.GetTestWorkflowTemplateRequest) (*cloud.GetTestWorkflowTemplateResponse, error) {
	template, err := s.testWorkflowTemplatesClient.Get(ctx, "", req.Name)
	if err != nil {
		return nil, err
	}
	templateBytes, err := json.Marshal(template)
	if err != nil {
		return nil, err
	}
	return &cloud.GetTestWorkflowTemplateResponse{Template: templateBytes}, nil
}

func (s *Server) ListTestWorkflowTemplates(req *cloud.ListTestWorkflowTemplatesRequest, srv cloud.TestKubeCloudAPI_ListTestWorkflowTemplatesServer) error {
	templates, err := s.testWorkflowTemplatesClient.List(srv.Context(), "", testworkflowtemplateclient.ListOptions{
		Labels:     req.Labels,
		TextSearch: req.TextSearch,
		Offset:     req.Offset,
		Limit:      req.Limit,
	})
	if err != nil {
		return err
	}
	var templateBytes []byte
	for _, template := range templates {
		templateBytes, err = json.Marshal(template)
		if err != nil {
			return err
		}
		err = srv.Send(&cloud.TestWorkflowTemplateListItem{Template: templateBytes})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) ListTestWorkflowTemplateLabels(ctx context.Context, req *cloud.ListTestWorkflowTemplateLabelsRequest) (*cloud.ListTestWorkflowTemplateLabelsResponse, error) {
	labels, err := s.testWorkflowTemplatesClient.ListLabels(ctx, "")
	if err != nil {
		return nil, err
	}
	res := &cloud.ListTestWorkflowTemplateLabelsResponse{Labels: make([]*cloud.LabelListItem, 0, len(labels))}
	for k, v := range labels {
		res.Labels = append(res.Labels, &cloud.LabelListItem{Name: k, Value: v})
	}
	return res, nil
}

func (s *Server) CreateTestWorkflowTemplate(ctx context.Context, req *cloud.CreateTestWorkflowTemplateRequest) (*cloud.CreateTestWorkflowTemplateResponse, error) {
	var template testkube.TestWorkflowTemplate
	err := json.Unmarshal(req.Template, &template)
	if err != nil {
		return nil, err
	}
	err = s.testWorkflowTemplatesClient.Create(ctx, "", template)
	if err != nil {
		return nil, err
	}
	return &cloud.CreateTestWorkflowTemplateResponse{}, nil
}

func (s *Server) UpdateTestWorkflowTemplate(ctx context.Context, req *cloud.UpdateTestWorkflowTemplateRequest) (*cloud.UpdateTestWorkflowTemplateResponse, error) {
	var template testkube.TestWorkflowTemplate
	err := json.Unmarshal(req.Template, &template)
	if err != nil {
		return nil, err
	}
	err = s.testWorkflowTemplatesClient.Update(ctx, "", template)
	if err != nil {
		return nil, err
	}
	return &cloud.UpdateTestWorkflowTemplateResponse{}, nil
}

func (s *Server) DeleteTestWorkflowTemplate(ctx context.Context, req *cloud.DeleteTestWorkflowTemplateRequest) (*cloud.DeleteTestWorkflowTemplateResponse, error) {
	err := s.testWorkflowTemplatesClient.Delete(ctx, "", req.Name)
	if err != nil {
		return nil, err
	}
	return &cloud.DeleteTestWorkflowTemplateResponse{}, nil
}

func (s *Server) DeleteTestWorkflowTemplatesByLabels(ctx context.Context, req *cloud.DeleteTestWorkflowTemplatesByLabelsRequest) (*cloud.DeleteTestWorkflowTemplatesByLabelsResponse, error) {
	count, err := s.testWorkflowTemplatesClient.DeleteByLabels(ctx, "", req.Labels)
	if err != nil {
		return nil, err
	}
	return &cloud.DeleteTestWorkflowTemplatesByLabelsResponse{Count: count}, nil
}

func (s *Server) FinishExecution(ctx context.Context, req *cloud.FinishExecutionRequest) (*cloud.FinishExecutionResponse, error) {
	var result testkube.TestWorkflowResult
	err := json.Unmarshal(req.Result, &result)
	if err != nil {
		return nil, err
	}
	err = s.resultsRepository.UpdateResult(ctx, req.Id, &result)
	if err != nil {
		return nil, err
	}
	return &cloud.FinishExecutionResponse{}, nil
}

func (s *Server) GetExecution(ctx context.Context, req *cloud.GetExecutionRequest) (*cloud.GetExecutionResponse, error) {
	execution, err := s.resultsRepository.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	executionBytes, err := json.Marshal(execution)
	if err != nil {
		return nil, err
	}
	return &cloud.GetExecutionResponse{Execution: executionBytes}, nil
}

func (s *Server) GetUnfinishedExecutions(_ *emptypb.Empty, srv cloud.TestKubeCloudAPI_GetUnfinishedExecutionsServer) error {
	executions, err := s.resultsRepository.GetExecutions(srv.Context(), testworkflow.FilterImpl{
		FStatuses: []testkube.TestWorkflowStatus{testkube.PAUSED_TestWorkflowStatus, testkube.QUEUED_TestWorkflowStatus, testkube.RUNNING_TestWorkflowStatus},
		FPageSize: math.MaxInt32,
	})
	if err != nil {
		return err
	}
	for _, execution := range executions {
		err = srv.Send(&cloud.UnfinishedExecution{Id: execution.Id})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) UpdateExecutionResult(ctx context.Context, req *cloud.UpdateExecutionResultRequest) (*cloud.UpdateExecutionResultResponse, error) {
	var result testkube.TestWorkflowResult
	err := json.Unmarshal(req.Result, &result)
	if err != nil {
		return nil, err
	}
	err = s.resultsRepository.UpdateResult(ctx, req.Id, &result)
	if err != nil {
		return nil, err
	}
	return &cloud.UpdateExecutionResultResponse{}, nil
}

func (s *Server) UpdateExecutionOutput(ctx context.Context, req *cloud.UpdateExecutionOutputRequest) (*cloud.UpdateExecutionOutputResponse, error) {
	err := s.resultsRepository.UpdateOutput(ctx, req.Id, common.MapSlice(req.Output, func(t *cloud.ExecutionOutput) testkube.TestWorkflowOutput {
		var v map[string]interface{}
		_ = json.Unmarshal(t.Value, &v)
		return testkube.TestWorkflowOutput{Ref: t.Ref, Name: t.Name, Value: v}
	}))
	if err != nil {
		return nil, err
	}
	return &cloud.UpdateExecutionOutputResponse{}, nil
}

func (s *Server) SaveExecutionLogsPresigned(ctx context.Context, req *cloud.SaveExecutionLogsPresignedRequest) (*cloud.SaveExecutionLogsPresignedResponse, error) {
	url, err := s.outputRepository.PresignSaveLog(ctx, req.Id, "")
	if err != nil {
		return nil, err
	}
	return &cloud.SaveExecutionLogsPresignedResponse{Url: url}, nil
}

func (s *Server) SaveExecutionArtifactPresigned(ctx context.Context, req *cloud.SaveExecutionArtifactPresignedRequest) (*cloud.SaveExecutionArtifactPresignedResponse, error) {
	url, err := s.storageClient.PresignUploadFileToBucket(ctx, s.cfg.StorageBucket, req.Id, req.FilePath, 15*time.Minute)
	if err != nil {
		return nil, err
	}
	return &cloud.SaveExecutionArtifactPresignedResponse{Url: url}, nil
}

func (s *Server) GetRepositoryManager() repository.DatabaseRepository {
	return s.repositoryManager
}
