package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2/middleware/cors"
	"google.golang.org/grpc"

	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	testkubeclientset "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/cmd/api-server/services"
	"github.com/kubeshop/testkube/internal/app/api/debug"
	"github.com/kubeshop/testkube/internal/app/api/oauth"
	cloudartifacts "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/event/kind/cdevent"
	"github.com/kubeshop/testkube/pkg/event/kind/k8sevent"
	"github.com/kubeshop/testkube/pkg/event/kind/webhook"
	ws "github.com/kubeshop/testkube/pkg/event/kind/websocket"
	oauth2 "github.com/kubeshop/testkube/pkg/oauth"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/tcl/checktcl"
	"github.com/kubeshop/testkube/pkg/tcl/schedulertcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/presets"

	"github.com/kubeshop/testkube/internal/common"
	parser "github.com/kubeshop/testkube/internal/template"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/version"

	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	kubeexecutor "github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/containerexecutor"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/scheduler"

	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/triggers"

	kubeclient "github.com/kubeshop/testkube-operator/pkg/client"
	testtriggersclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testtriggers/v1"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	deprecatedapiv1 "github.com/kubeshop/testkube/internal/app/api/deprecatedv1"
	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/reconciler"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

func init() {
	flag.Parse()
}

func main() {
	cfg := commons.MustGetConfig()
	features := commons.MustGetFeatureFlags()

	// Determine the running mode
	mode := common.ModeStandalone
	if cfg.TestkubeProAPIKey != "" {
		mode = common.ModeAgent
	}

	// Run services within an errgroup to propagate errors between services.
	g, ctx := errgroup.WithContext(context.Background())

	// Cancel the errgroup context on SIGINT and SIGTERM,
	// which shuts everything down gracefully.
	g.Go(commons.HandleCancelSignal(ctx))

	commons.MustFreePort(cfg.APIServerPort)
	commons.MustFreePort(cfg.GraphqlPort)
	commons.MustFreePort(cfg.GRPCServerPort)

	configMapConfig := commons.MustGetConfigMapConfig(ctx, cfg.APIServerConfig, cfg.TestkubeNamespace, cfg.TestkubeAnalyticsEnabled)

	// Start local Control Plane
	if mode == common.ModeStandalone {
		controlPlane := services.CreateControlPlane(ctx, cfg, features, configMapConfig, true)
		g.Go(func() error {
			return controlPlane.Run(ctx)
		})

		// Rewire connection
		cfg.TestkubeProURL = fmt.Sprintf("%s:%d", cfg.APIServerFullname, cfg.GRPCServerPort)
		cfg.TestkubeProTLSInsecure = true
	}

	clusterId, _ := configMapConfig.GetUniqueClusterId(ctx)
	telemetryEnabled, _ := configMapConfig.GetTelemetryEnabled(ctx)

	// k8s
	kubeClient, err := kubeclient.GetClient()
	commons.ExitOnError("Getting kubernetes client", err)
	clientset, err := k8sclient.ConnectToK8s()
	commons.ExitOnError("Creating k8s clientset", err)

	// k8s clients
	secretClient := secret.NewClientFor(clientset, cfg.TestkubeNamespace)
	configMapClient := configmap.NewClientFor(clientset, cfg.TestkubeNamespace)
	deprecatedClients := commons.CreateDeprecatedClients(kubeClient, cfg.TestkubeNamespace)
	webhooksClient := executorsclientv1.NewWebhooksClient(kubeClient, cfg.TestkubeNamespace)
	testTriggersClient := testtriggersclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)
	testWorkflowExecutionsClient := testworkflowsclientv1.NewTestWorkflowExecutionsClient(kubeClient, cfg.TestkubeNamespace)

	// TODO: Make granular environment variables, yet backwards compatible
	secretConfig := testkube.SecretConfig{
		Prefix:     cfg.SecretCreationPrefix,
		List:       cfg.EnableSecretsEndpoint,
		ListAll:    cfg.EnableSecretsEndpoint && cfg.EnableListingAllSecrets,
		Create:     cfg.EnableSecretsEndpoint && !cfg.DisableSecretCreation,
		Modify:     cfg.EnableSecretsEndpoint && !cfg.DisableSecretCreation,
		Delete:     cfg.EnableSecretsEndpoint && !cfg.DisableSecretCreation,
		AutoCreate: !cfg.DisableSecretCreation,
	}
	secretManager := secretmanager.New(clientset, secretConfig)

	envs := commons.GetEnvironmentVariables()

	defaultExecutors, images, err := commons.ReadDefaultExecutors(cfg)
	commons.ExitOnError("Parsing default executors", err)
	if !cfg.TestkubeReadonlyExecutors {
		err := kubeexecutor.SyncDefaultExecutors(deprecatedClients.Executors(), cfg.TestkubeNamespace, defaultExecutors)
		commons.ExitOnError("Sync default executors", err)
	}
	jobTemplates, err := parser.ParseJobTemplates(cfg)
	commons.ExitOnError("Creating job templates", err)
	containerTemplates, err := parser.ParseContainerTemplates(cfg)
	commons.ExitOnError("Creating container job templates", err)

	inspector := commons.CreateImageInspector(cfg, configMapClient, secretClient)

	var testWorkflowsClient testworkflowsclientv1.Interface
	var testWorkflowTemplatesClient testworkflowsclientv1.TestWorkflowTemplatesInterface

	var grpcClient cloud.TestKubeCloudAPIClient
	var grpcConn *grpc.ClientConn
	// Use local network for local access
	controlPlaneUrl := cfg.TestkubeProURL
	if strings.HasPrefix(controlPlaneUrl, fmt.Sprintf("%s:%d", cfg.APIServerFullname, cfg.GRPCServerPort)) {
		controlPlaneUrl = fmt.Sprintf("127.0.0.1:%d", cfg.GRPCServerPort)
	}
	grpcConn, err = agent.NewGRPCConnection(
		ctx,
		cfg.TestkubeProTLSInsecure,
		cfg.TestkubeProSkipVerify,
		controlPlaneUrl,
		cfg.TestkubeProCertFile,
		cfg.TestkubeProKeyFile,
		cfg.TestkubeProCAFile, //nolint
		log.DefaultLogger,
	)
	commons.ExitOnError("error creating gRPC connection", err)

	grpcClient = cloud.NewTestKubeCloudAPIClient(grpcConn)

	if mode == common.ModeAgent && cfg.WorkflowStorage == "control-plane" {
		testWorkflowsClient = cloudtestworkflow.NewCloudTestWorkflowRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
		testWorkflowTemplatesClient = cloudtestworkflow.NewCloudTestWorkflowTemplateRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
	} else {
		testWorkflowsClient = testworkflowsclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)
		testWorkflowTemplatesClient = testworkflowsclientv1.NewTestWorkflowTemplatesClient(kubeClient, cfg.TestkubeNamespace)
	}

	deprecatedRepositories := commons.CreateDeprecatedRepositoriesForCloud(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
	testWorkflowResultsRepository := cloudtestworkflow.NewCloudRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
	var opts []cloudtestworkflow.Option
	if cfg.StorageSkipVerify {
		opts = append(opts, cloudtestworkflow.WithSkipVerify())
	}
	testWorkflowOutputRepository := cloudtestworkflow.NewCloudOutputRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey, opts...)
	triggerLeaseBackend := triggers.NewAcquireAlwaysLeaseBackend()
	artifactStorage := cloudartifacts.NewCloudArtifactsStorage(grpcClient, grpcConn, cfg.TestkubeProAPIKey)

	nc := commons.MustCreateNATSConnection(cfg)
	eventBus := bus.NewNATSBus(nc)
	if cfg.Trace {
		eventBus.TraceEvents()
	}
	eventsEmitter := event.NewEmitter(eventBus, cfg.TestkubeClusterName)

	var logGrpcClient logsclient.StreamGetter
	var logsStream logsclient.Stream
	if features.LogsV2 {
		logGrpcClient = commons.MustGetLogsV2Client(cfg)
		logsStream, err = logsclient.NewNatsLogStream(nc.Conn)
		commons.ExitOnError("Creating logs streaming client", err)
	}

	// Check Pro/Enterprise subscription
	proContext := commons.ReadProContext(ctx, cfg, grpcClient)
	subscriptionChecker, err := checktcl.NewSubscriptionChecker(ctx, proContext, grpcClient, grpcConn)
	commons.ExitOnError("Failed creating subscription checker", err)

	serviceAccountNames := map[string]string{
		cfg.TestkubeNamespace: cfg.JobServiceAccountName,
	}
	// Pro edition only (tcl protected code)
	if cfg.TestkubeExecutionNamespaces != "" {
		err = subscriptionChecker.IsActiveOrgPlanEnterpriseForFeature("execution namespace")
		commons.ExitOnError("Subscription checking", err)
		serviceAccountNames = schedulertcl.GetServiceAccountNamesFromConfig(serviceAccountNames, cfg.TestkubeExecutionNamespaces)
	}

	metrics := metrics.NewMetrics()

	executor, err := client.NewJobExecutor(
		deprecatedRepositories,
		deprecatedClients,
		images,
		jobTemplates,
		serviceAccountNames,
		metrics,
		eventsEmitter,
		configMapConfig,
		clientset,
		cfg.TestkubeRegistry,
		cfg.TestkubePodStartTimeout,
		clusterId,
		cfg.TestkubeDashboardURI,
		fmt.Sprintf("http://%s:%d", cfg.APIServerFullname, cfg.APIServerPort),
		cfg.NatsURI,
		cfg.Debug,
		logsStream,
		features,
		cfg.TestkubeDefaultStorageClassName,
		cfg.WhitelistedContainers,
	)
	commons.ExitOnError("Creating executor client", err)

	containerExecutor, err := containerexecutor.NewContainerExecutor(
		deprecatedRepositories,
		deprecatedClients,
		images,
		containerTemplates,
		inspector,
		serviceAccountNames,
		metrics,
		eventsEmitter,
		configMapConfig,
		cfg.TestkubeRegistry,
		cfg.TestkubePodStartTimeout,
		clusterId,
		cfg.TestkubeDashboardURI,
		fmt.Sprintf("http://%s:%d", cfg.APIServerFullname, cfg.APIServerPort),
		cfg.NatsURI,
		cfg.Debug,
		logsStream,
		features,
		cfg.TestkubeDefaultStorageClassName,
		cfg.WhitelistedContainers,
		cfg.TestkubeImageCredentialsCacheTTL,
	)
	commons.ExitOnError("Creating container executor", err)

	sched := scheduler.NewScheduler(
		metrics,
		executor,
		containerExecutor,
		deprecatedRepositories,
		deprecatedClients,
		secretClient,
		eventsEmitter,
		log.DefaultLogger,
		configMapConfig,
		configMapClient,
		eventBus,
		cfg.TestkubeDashboardURI,
		features,
		logsStream,
		cfg.TestkubeNamespace,
		cfg.TestkubeProTLSSecret,
		cfg.TestkubeProRunnerCustomCASecret,
		subscriptionChecker,
	)

	// Build internal execution worker
	testWorkflowProcessor := presets.NewOpenSource(inspector)
	// Pro edition only (tcl protected code)
	if mode == common.ModeAgent {
		testWorkflowProcessor = presets.NewPro(inspector)
	}
	executionWorker := services.CreateExecutionWorker(clientset, cfg, clusterId, serviceAccountNames, testWorkflowProcessor)

	testWorkflowExecutor := testworkflowexecutor.New(
		eventsEmitter,
		executionWorker,
		clientset,
		testWorkflowResultsRepository,
		testWorkflowOutputRepository,
		configMapConfig,
		testWorkflowTemplatesClient,
		testWorkflowExecutionsClient,
		testWorkflowsClient,
		metrics,
		secretManager,
		cfg.GlobalWorkflowTemplateName,
		cfg.TestkubeDashboardURI,
		&proContext,
	)
	g.Go(func() error {
		testWorkflowExecutor.Recover(ctx)
		return nil
	})

	// Initialize event handlers
	websocketLoader := ws.NewWebsocketLoader()
	eventsEmitter.Loader.Register(webhook.NewWebhookLoader(log.DefaultLogger, webhooksClient, deprecatedClients.Templates(), deprecatedRepositories.TestResults(), deprecatedRepositories.TestSuiteResults(), testWorkflowResultsRepository, metrics, &proContext, envs))
	eventsEmitter.Loader.Register(websocketLoader)
	eventsEmitter.Loader.Register(commons.MustCreateSlackLoader(cfg, envs))
	if cfg.CDEventsTarget != "" {
		cdeventLoader, err := cdevent.NewCDEventLoader(cfg.CDEventsTarget, clusterId, cfg.TestkubeNamespace, cfg.TestkubeDashboardURI, testkube.AllEventTypes)
		if err == nil {
			eventsEmitter.Loader.Register(cdeventLoader)
		} else {
			log.DefaultLogger.Debugw("cdevents init error", "error", err.Error())
		}
	}
	if cfg.EnableK8sEvents {
		eventsEmitter.Loader.Register(k8sevent.NewK8sEventLoader(clientset, cfg.TestkubeNamespace, testkube.AllEventTypes))
	}
	eventsEmitter.Listen(ctx)
	g.Go(func() error {
		eventsEmitter.Reconcile(ctx)
		return nil
	})

	// Create HTTP server
	httpServer := server.NewServer(server.Config{Port: cfg.APIServerPort})
	httpServer.Routes.Use(cors.New())

	// Handle OAuth TODO: deprecated?
	httpServer.Routes.Use(oauth.CreateOAuthHandler(oauth.OauthParams{
		ClientID:     cfg.TestkubeOAuthClientID,
		ClientSecret: cfg.TestkubeOAuthClientSecret,
		Provider:     oauth2.ProviderType(cfg.TestkubeOAuthProvider),
		Scopes:       cfg.TestkubeOAuthScopes,
	}))

	storageParams := deprecatedapiv1.StorageParams{
		SSL:             cfg.StorageSSL,
		SkipVerify:      cfg.StorageSkipVerify,
		CertFile:        cfg.StorageCertFile,
		KeyFile:         cfg.StorageKeyFile,
		CAFile:          cfg.StorageCAFile,
		Endpoint:        cfg.StorageEndpoint,
		AccessKeyId:     cfg.StorageAccessKeyID,
		SecretAccessKey: cfg.StorageSecretAccessKey,
		Region:          cfg.StorageRegion,
		Token:           cfg.StorageToken,
		Bucket:          cfg.StorageBucket,
	}
	// Use direct MinIO artifact storage for deprecated API for backwards compatibility
	deprecatedArtifactStorage := storage.ArtifactsStorage(artifactStorage)
	if mode == common.ModeStandalone {
		deprecatedArtifactStorage = minio.NewMinIOArtifactClient(commons.MustGetMinioClient(cfg))
	}
	deprecatedApi := deprecatedapiv1.NewDeprecatedTestkubeAPI(
		deprecatedRepositories,
		deprecatedClients,
		cfg.TestkubeNamespace,
		secretClient,
		eventsEmitter,
		executor,
		containerExecutor,
		metrics,
		sched,
		cfg.GraphqlPort,
		deprecatedArtifactStorage,
		mode,
		eventBus,
		secretConfig,
		features,
		logsStream,
		logGrpcClient,
		&proContext,
		storageParams,
	)
	deprecatedApi.Init(httpServer)

	api := apiv1.NewTestkubeAPI(
		deprecatedClients,
		clusterId,
		cfg.TestkubeNamespace,
		testWorkflowResultsRepository,
		testWorkflowOutputRepository,
		artifactStorage,
		webhooksClient,
		testTriggersClient,
		testWorkflowsClient,
		testWorkflowTemplatesClient,
		configMapConfig,
		secretManager,
		secretConfig,
		testWorkflowExecutor,
		executionWorker,
		eventsEmitter,
		websocketLoader,
		metrics,
		&proContext,
		features,
		cfg.TestkubeDashboardURI,
		cfg.TestkubeHelmchartVersion,
		serviceAccountNames,
		cfg.TestkubeDockerImageVersion,
	)
	api.Init(httpServer)

	log.DefaultLogger.Info("starting agent service")
	getTestWorkflowNotificationsStream := func(ctx context.Context, executionID string) (<-chan testkube.TestWorkflowExecutionNotification, error) {
		execution, err := testWorkflowResultsRepository.Get(ctx, executionID)
		if err != nil {
			return nil, err
		}
		notifications := executionWorker.Notifications(ctx, execution.Id, executionworkertypes.NotificationsOptions{
			Hints: executionworkertypes.Hints{
				Namespace:   execution.Namespace,
				Signature:   execution.Signature,
				ScheduledAt: common.Ptr(execution.ScheduledAt),
			},
		})
		if notifications.Err() != nil {
			return nil, notifications.Err()
		}
		return notifications.Channel(), nil
	}
	agentHandle, err := agent.NewAgent(
		log.DefaultLogger,
		httpServer.Mux.Handler(),
		grpcClient,
		deprecatedApi.GetLogsStream,
		getTestWorkflowNotificationsStream,
		clusterId,
		cfg.TestkubeClusterName,
		features,
		&proContext,
		cfg.TestkubeDockerImageVersion,
	)
	commons.ExitOnError("Starting agent", err)
	g.Go(func() error {
		err = agentHandle.Run(ctx)
		commons.ExitOnError("Running agent", err)
		return nil
	})
	eventsEmitter.Loader.Register(agentHandle)

	if !cfg.DisableTestTriggers {
		k8sCfg, err := k8sclient.GetK8sClientConfig()
		commons.ExitOnError("Getting k8s client config", err)
		testkubeClientset, err := testkubeclientset.NewForConfig(k8sCfg)
		commons.ExitOnError("Creating TestKube Clientset", err)
		// TODO: Check why this simpler options is not working
		//testkubeClientset := testkubeclientset.New(clientset.RESTClient())

		triggerService := triggers.NewService(
			deprecatedRepositories,
			deprecatedClients,
			sched,
			clientset,
			testkubeClientset,
			testWorkflowsClient,
			triggerLeaseBackend,
			log.DefaultLogger,
			configMapConfig,
			executor,
			eventBus,
			metrics,
			executionWorker,
			testWorkflowExecutor,
			testWorkflowResultsRepository,
			triggers.WithHostnameIdentifier(),
			triggers.WithTestkubeNamespace(cfg.TestkubeNamespace),
			triggers.WithWatcherNamespaces(cfg.TestkubeWatcherNamespaces),
			triggers.WithDisableSecretCreation(!secretConfig.AutoCreate),
		)
		log.DefaultLogger.Info("starting trigger service")
		g.Go(func() error {
			triggerService.Run(ctx)
			return nil
		})
	} else {
		log.DefaultLogger.Info("test triggers are disabled")
	}

	if !cfg.DisableReconciler {
		reconcilerClient := reconciler.NewClient(clientset, deprecatedRepositories, deprecatedClients, log.DefaultLogger)
		g.Go(func() error {
			return reconcilerClient.Run(ctx)
		})
	} else {
		log.DefaultLogger.Info("reconciler is disabled")
	}

	// telemetry based functions
	g.Go(func() error {
		services.HandleTelemetryHeartbeat(ctx, clusterId, configMapConfig)
		return nil
	})

	log.DefaultLogger.Infow(
		"starting Testkube API server",
		"telemetryEnabled", telemetryEnabled,
		"clusterId", clusterId,
		"namespace", cfg.TestkubeNamespace,
		"version", version.Version,
	)

	if cfg.EnableDebugServer {
		debugSrv := debug.NewDebugServer(cfg.DebugListenAddr)

		g.Go(func() error {
			log.DefaultLogger.Infof("starting debug pprof server")
			return debugSrv.ListenAndServe()
		})
	}

	g.Go(func() error {
		return httpServer.Run(ctx)
	})

	g.Go(func() error {
		return deprecatedApi.RunGraphQLServer(ctx)
	})

	if err := g.Wait(); err != nil {
		log.DefaultLogger.Fatalf("Testkube is shutting down: %v", err)
	}
}
