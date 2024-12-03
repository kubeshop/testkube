package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	errors2 "errors"
	"fmt"
	"math"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2/log"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/expressions"
	log2 "github.com/kubeshop/testkube/pkg/log"
	testworkflowmappers "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

const (
	KeepAliveTime       = 10 * time.Second
	KeepAliveTimeout    = 5 * time.Second
	HealthCheckInterval = 60 * time.Second
	SendPingInterval    = HealthCheckInterval / 2
)

type Server struct {
	cloud.UnimplementedTestKubeCloudAPIServer
	cfg          Config
	server       *grpc.Server
	scheduler    *testworkflowexecutor.ExecutionScheduler
	dashboardUri string
	commands     map[executor.Command]CommandHandler
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
		cfg:       cfg,
		scheduler: executionScheduler,
		commands:  commands,
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

// TODO: Limit selectors or maximum executions to avoid huge load?
func (s *Server) ScheduleExecution(req *cloud.ScheduleRequest, srv cloud.TestKubeCloudAPI_ScheduleExecutionServer) error {
	// Validate if there is anything to run
	if len(req.Selectors) == 0 {
		return nil
	}

	// Validate if the selectors have exclusively name or label selector
	nameSelectorsCount := 0
	labelSelectorsCount := 0
	for i := range req.Selectors {
		if req.Selectors[i] == nil {
			return errors.New("invalid selector provided")
		}
		if req.Selectors[i].Name != "" && len(req.Selectors[i].LabelSelector) > 0 {
			return errors.New("invalid selector provided")
		}
		if req.Selectors[i].Name == "" && len(req.Selectors[i].LabelSelector) == 0 {
			return errors.New("invalid selector provided")
		}
		if req.Selectors[i].Name != "" {
			nameSelectorsCount++
		} else {
			labelSelectorsCount++
		}
	}

	// Validate if that could be Kubernetes object
	if req.KubernetesObjectName != "" && (nameSelectorsCount != 1 || labelSelectorsCount != 0) {
		return errors.New("kubernetes object can trigger only execution of a single named TestWorkflow")
	}

	// Get dependencies
	// TODO: IntermediateExecutionSet
	testWorkflowsClient := s.scheduler.TestWorkflowsClient()
	testWorkflowTemplatesClient := s.scheduler.TestWorkflowsTemplatesClient()
	resultsRepository := s.scheduler.Repository()
	outputRepository := s.scheduler.OutputRepository()
	emitter := s.scheduler.Emitter()
	runner := s.scheduler.Runner()
	secretManager := s.scheduler.SecretManager()
	globalTemplateName := s.scheduler.GlobalTemplateName()

	// Prepare multi-get
	testWorkflowCache := make(map[string]*testworkflowsv1.TestWorkflow)
	testWorkflowTemplateCache := make(map[string]testworkflowsv1.TestWorkflowTemplate)
	getTestWorkflow := func(name string) (*testworkflowsv1.TestWorkflow, error) {
		if v, ok := testWorkflowCache[name]; ok {
			return v.DeepCopy(), nil
		}
		workflow, err := testWorkflowsClient.Get(name)
		if err != nil {
			return nil, err
		}
		testWorkflowCache[name] = workflow.DeepCopy()
		return workflow, nil
	}
	getTestWorkflows := func(labels map[string]string) ([]testworkflowsv1.TestWorkflow, error) {
		selectors := make([]string, 0, len(labels))
		for k := range labels {
			selectors = append(selectors, fmt.Sprintf("%s=%s", k, labels[k]))
		}
		workflows, err := testWorkflowsClient.List(strings.Join(selectors, ","))
		if err != nil {
			return nil, err
		}
		for i := range workflows.Items {
			testWorkflowCache[workflows.Items[i].Name] = &workflows.Items[i]
		}
		return workflows.Items, nil
	}
	getTestWorkflowTemplates := func(names map[string]struct{}) (map[string]testworkflowsv1.TestWorkflowTemplate, error) {
		left := make(map[string]struct{})
		result := make(map[string]testworkflowsv1.TestWorkflowTemplate)

		// Get from cache
		for name := range names {
			if v, ok := testWorkflowTemplateCache[name]; ok {
				result[name] = v
			} else {
				left[name] = struct{}{}
			}
		}

		// Load missing TODO: parallel
		for name := range left {
			tpl, err := testWorkflowTemplatesClient.Get(name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get test workflow template '%s'", testworkflowresolver.GetDisplayTemplateName(name))
			}
			testWorkflowTemplateCache[name] = *tpl
			result[name] = *tpl
		}
		return result, nil
	}
	isExecutionNameReserved := func(ctx context.Context, name, workflowName string) (bool, error) {
		// TODO: Detect errors other than 404?
		next, _ := resultsRepository.GetByNameAndTestWorkflow(ctx, name, workflowName)
		if next.Name == name {
			return true, nil
		}
		return false, nil
	}
	insert := func(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
		return retry(testworkflowexecutor.SaveResultRetryMaxAttempts, testworkflowexecutor.SaveResultRetryBaseDelay, func() error {
			return resultsRepository.Insert(context.Background(), *execution)
		})
	}
	update := func(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
		return retry(testworkflowexecutor.SaveResultRetryMaxAttempts, testworkflowexecutor.SaveResultRetryBaseDelay, func() error {
			return resultsRepository.Update(context.Background(), *execution)
		})
	}
	init := func(ctx context.Context, execution *testkube.TestWorkflowExecution) error {
		return retry(testworkflowexecutor.SaveResultRetryMaxAttempts, testworkflowexecutor.SaveResultRetryBaseDelay, func() error {
			return resultsRepository.Init(context.Background(), execution.Id, testworkflow.InitData{
				RunnerID:  execution.RunnerId,
				Namespace: execution.Namespace,
				Signature: execution.Signature,
			})
		})
	}
	saveEmptyLogs := func(execution *testkube.TestWorkflowExecution) {
		err := retry(testworkflowexecutor.SaveResultRetryMaxAttempts, testworkflowexecutor.SaveResultRetryBaseDelay, func() error {
			return outputRepository.SaveLog(context.Background(), execution.Id, execution.Workflow.Name, bytes.NewReader(nil))
		})
		if err != nil {
			log2.DefaultLogger.Errorw("failed to save empty log", "executionId", execution.Id, "error", err)
		}
	}

	// Set up context
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	// Set up basic data
	now := time.Now().UTC()
	groupId := primitive.NewObjectIDFromTimestamp(now).Hex()

	// Translate to the old format
	runningContext := GetLegacyRunningContext(req)

	// Initialize execution template
	base := testworkflowexecutor.NewIntermediateExecution().
		SetGroupID(groupId).
		SetScheduledAt(now).
		AppendTags(req.Tags).
		SetDisabledWebhooks(req.DisableWebhooks).
		SetKubernetesObjectName(req.KubernetesObjectName).
		SetRunningContext(runningContext).
		PrependTemplate(globalTemplateName)

	// Load the required Test Workflows and build the basic selector
	nameSelectors := make([]*cloud.ScheduleSelector, 0, len(req.Selectors))
	for i := range req.Selectors {
		if req.Selectors[i].Name != "" {
			// Get the Test Workflow from the storage
			_, err := getTestWorkflow(req.Selectors[i].Name)
			if err != nil {
				return errors.Wrapf(err, "cannot get workflow '%s'", req.Selectors[i].Name)
			}

			// Add a selector back
			nameSelectors = append(nameSelectors, req.Selectors[i])
		} else {
			// Get the matching Test Workflows from the storage
			workflows, err := getTestWorkflows(req.Selectors[i].LabelSelector)
			if err != nil {
				return errors.Wrapf(err, "cannot get workflow for selector '%v'", req.Selectors[i].LabelSelector)
			}

			// Add a selector for each of the found Test Workflows
			for j := range workflows {
				nameSelectors = append(nameSelectors, &cloud.ScheduleSelector{
					Name:          workflows[j].Name,
					Config:        req.Selectors[i].Config,
					ExecutionName: req.Selectors[i].ExecutionName, // TODO: what to do when execution name is configured, but multiple requested?
					Tags:          req.Selectors[i].Tags,
				})
			}
		}
	}

	// Resolve executions for each selector
	intermediate := make([]*testworkflowexecutor.IntermediateExecution, 0, len(req.Selectors))
	for _, v := range req.Selectors {
		current := base.Clone().
			AutoGenerateID().
			SetName(v.ExecutionName).
			AppendTags(v.Tags).
			SetWorkflow(testWorkflowCache[v.Name])
		intermediate = append(intermediate, current)

		// Apply the configuration
		if err := current.ApplyConfig(v.Config); err != nil {
			current.SetError("Cannot inline Test Workflow configuration", err)
			continue
		}

		// List the Test Workflow Templates to fetch
		tpls, err := getTestWorkflowTemplates(current.TemplateNames())
		if err != nil {
			current.SetError("Cannot fetch required Test Workflow Templates", err)
			continue
		}

		// Apply the Test Workflow Templates
		if err = current.ApplyTemplates(tpls); err != nil {
			current.SetError("Cannot inline Test Workflow Templates", err)
			continue
		}
	}

	// Simplify group ID in case of single execution
	if len(intermediate) == 1 {
		intermediate[0].SetGroupID(intermediate[0].ID())
	}

	// Validate if there are no duplicated execution names in the set
	type namePair struct {
		Workflow  string
		Execution string
	}
	localDuplicatesCheck := make(map[namePair]struct{})
	for i := range intermediate {
		if intermediate[i].Name() == "" {
			continue
		}
		key := namePair{Workflow: intermediate[i].WorkflowName(), Execution: intermediate[i].Name()}
		if _, ok := localDuplicatesCheck[key]; ok {
			return fmt.Errorf("duplicated execution name: '%s' for workflow '%s'", intermediate[i].Name(), intermediate[i].WorkflowName())
		}
		localDuplicatesCheck[key] = struct{}{}
	}

	// Validate if the static execution names are not reserved in the database already
	for i := range intermediate {
		if intermediate[i].Name() == "" {
			continue
		}
		reserved, err := isExecutionNameReserved(ctx, intermediate[i].Name(), intermediate[i].WorkflowName())
		if err != nil {
			return errors.Wrapf(err, "checking for unique name: '%s' for workflow '%s'", intermediate[i].Name(), intermediate[i].WorkflowName())
		}
		if reserved {
			return fmt.Errorf("execution name already exists: '%s' for workflow '%s'", intermediate[i].Name(), intermediate[i].WorkflowName())
		}
	}

	// Generate execution names and sequence numbers for each execution
	for i := range intermediate {
		// Load execution identifier data
		number, err := resultsRepository.GetNextExecutionNumber(context.Background(), intermediate[i].WorkflowName())
		if err != nil {
			return errors.Wrap(err, "registering next execution sequence number")
		}
		intermediate[i].SetSequenceNumber(number)

		// Generating the execution name
		if intermediate[i].Name() == "" {
			name := fmt.Sprintf("%s-%d", intermediate[i].WorkflowName(), number)
			intermediate[i].SetName(name)

			// Edge case: Check for local duplicates, if there is no clash between static and auto-generated one
			key := namePair{Workflow: intermediate[i].WorkflowName(), Execution: intermediate[i].Name()}
			if _, ok := localDuplicatesCheck[key]; ok {
				return fmt.Errorf("duplicated execution name: '%s' for workflow '%s'", intermediate[i].Name(), intermediate[i].WorkflowName())
			}

			// Ensure the execution name is unique
			reserved, err := isExecutionNameReserved(ctx, intermediate[i].Name(), intermediate[i].WorkflowName())
			if err != nil {
				return errors.Wrapf(err, "checking for unique name: '%s' for workflow '%s'", intermediate[i].Name(), intermediate[i].WorkflowName())
			}
			if reserved {
				return fmt.Errorf("execution name already exists: '%s' for workflow '%s'", intermediate[i].Name(), intermediate[i].WorkflowName())
			}
		}

		// Resolve it finally
		err = intermediate[i].Resolve("", "", req.ParentExecutionIds, false)
		if err != nil {
			intermediate[i].SetError("Cannot process Test Workflow specification", err)
			continue
		}
	}

	controlPlaneConfig := testworkflowconfig.ControlPlaneConfig{
		DashboardUrl:   s.dashboardUri,
		CDEventsTarget: os.Getenv("CDEVENTS_TARGET"),
	}

	// Ensure the rest of operations won't be stopped if stated
	if ctx.Err() != nil {
		return context.Canceled
	}
	cancel()
	ctx = context.Background()

	// Store in the database
	for i := range intermediate {
		// OSS: Prepare the sensitive data
		// TODO: Do it with actual credentials instead
		secretsBatch := secretManager.Batch("twe-", intermediate[i].ID()).ForceEnable()
		credentialExpressions := map[string]expressions.Expression{}
		for k, v := range intermediate[i].SensitiveData() {
			envVarSource, err := secretsBatch.Append(k, v)
			if err != nil {
				intermediate[i].SetError("Cannot store the sensitive data", err)
			}
			credentialExpressions[k] = expressions.MustCompile(fmt.Sprintf(`secret("%s","%s",true)`, envVarSource.SecretKeyRef.Name, envVarSource.SecretKeyRef.Key))
		}
		secrets := secretsBatch.Get()
		secretsMap := make(map[string]map[string]string, len(secrets))
		for j := range secrets {
			secretsMap[secrets[j].Name] = secrets[j].StringData
		}
		err := intermediate[i].RewriteSensitiveDataCall(func(name string) (expressions.Expression, error) {
			if expr, ok := credentialExpressions[name]; ok {
				return expr, nil
			}
			return nil, fmt.Errorf(`unknown sensitive data: '%s'`, name)
		})
		if err != nil {
			intermediate[i].SetError("Cannot access the sensitive data", err)
		}

		// Insert the execution
		err = insert(ctx, intermediate[i].Execution())
		if err != nil {
			// TODO: delete the credentials left-overs
			// TODO: don't fail immediately (try creating other executions too)
			return errors.Wrapf(err, "failed to insert execution '%s' in workflow '%s'", intermediate[i].ID(), intermediate[i].WorkflowName())
		}
		exec := intermediate[i].Execution()
		emitter.Notify(testkube.NewEventQueueTestWorkflow(exec))

		// Send the data
		v, err := json.Marshal(exec)
		if err != nil {
			return err
		}
		err = srv.Send(&cloud.ScheduleResponse{Execution: v})
		if err != nil {
			// TODO: retry?
		}

		// Finish early if it's immediately known to finish
		if intermediate[i].Finished() {
			emitter.Notify(testkube.NewEventStartTestWorkflow(exec))
			if exec.Result.IsAborted() {
				emitter.Notify(testkube.NewEventEndTestWorkflowAborted(exec))
			} else if exec.Result.IsFailed() {
				emitter.Notify(testkube.NewEventEndTestWorkflowFailed(exec))
			} else {
				emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(exec))
			}
			saveEmptyLogs(exec)
			continue
		}

		// Start the execution
		result, err := runner.Execute(executionworkertypes.ExecuteRequest{
			Execution: testworkflowconfig.ExecutionConfig{
				Id:              exec.Id,
				GroupId:         exec.GroupId,
				Name:            exec.Name,
				Number:          exec.Number,
				ScheduledAt:     exec.ScheduledAt,
				DisableWebhooks: exec.DisableWebhooks,
				Debug:           false,
				OrganizationId:  "",
				EnvironmentId:   "",
				ParentIds:       strings.Join(req.ParentExecutionIds, "/"),
			},
			Secrets:      secretsMap,
			Workflow:     testworkflowmappers.MapTestWorkflowAPIToKube(*exec.ResolvedWorkflow),
			ControlPlane: controlPlaneConfig,
		})

		// TODO: define "revoke" error by runner (?)
		if err != nil {
			exec.InitializationError("Failed to run execution", err)
			err2 := update(ctx, exec)
			err = errors2.Join(err, err2)
			if err != nil {
				log2.DefaultLogger.Errorw("failed to run and update execution", "executionId", exec.Id, "error", err)
			}

			emitter.Notify(testkube.NewEventStartTestWorkflow(exec))
			emitter.Notify(testkube.NewEventEndTestWorkflowAborted(exec))
			saveEmptyLogs(exec)
			continue
		}

		// Inform about execution start
		emitter.Notify(testkube.NewEventStartTestWorkflow(exec))

		// Apply the known data to temporary object.
		exec.Namespace = result.Namespace
		exec.Signature = result.Signature
		exec.Result.Steps = stage.MapSignatureListToStepResults(stage.MapSignatureList(result.Signature))
		if err = init(ctx, exec); err != nil {
			log2.DefaultLogger.Errorw("failed to mark execution as initialized", "executionId", exec.Id, "error", err)
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
