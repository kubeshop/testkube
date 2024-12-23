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
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	cloudartifacts "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/event/kind/cdevent"
	"github.com/kubeshop/testkube/pkg/event/kind/k8sevent"
	"github.com/kubeshop/testkube/pkg/event/kind/webhook"
	ws "github.com/kubeshop/testkube/pkg/event/kind/websocket"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/kubeshop/testkube/pkg/tcl/checktcl"
	"github.com/kubeshop/testkube/pkg/tcl/schedulertcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/presets"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/version"

	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/triggers"

	kubeclient "github.com/kubeshop/testkube-operator/pkg/client"
	testtriggersclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testtriggers/v1"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/log"
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
		controlPlane := services.CreateControlPlane(ctx, cfg, features, configMapConfig)
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
	grpcConn, err = agentclient.NewGRPCConnection(
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

	var deprecatedSystem *services.DeprecatedSystem
	if !cfg.DisableDeprecatedTests {
		deprecatedSystem = services.CreateDeprecatedSystem(
			ctx,
			mode,
			cfg,
			features,
			metrics,
			configMapConfig,
			secretConfig,
			grpcClient,
			grpcConn,
			nc,
			eventsEmitter,
			eventBus,
			inspector,
			subscriptionChecker,
			&proContext,
		)
	}

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

	var deprecatedClients commons.DeprecatedClients
	var deprecatedRepositories commons.DeprecatedRepositories
	if deprecatedSystem != nil {
		deprecatedClients = deprecatedSystem.Clients
		deprecatedRepositories = deprecatedSystem.Repositories
	}

	// Initialize event handlers
	websocketLoader := ws.NewWebsocketLoader()
	if !cfg.DisableWebhooks {
		eventsEmitter.Loader.Register(webhook.NewWebhookLoader(log.DefaultLogger, webhooksClient, deprecatedClients, deprecatedRepositories, testWorkflowResultsRepository, metrics, &proContext, envs))
	}
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

	if deprecatedSystem != nil && deprecatedSystem.API != nil {
		deprecatedSystem.API.Init(httpServer)
	}

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

	getDeprecatedLogStream := agent.GetDeprecatedLogStream
	if deprecatedSystem != nil && deprecatedSystem.StreamLogs != nil {
		getDeprecatedLogStream = deprecatedSystem.StreamLogs
	}
	agentHandle, err := agent.NewAgent(
		log.DefaultLogger,
		httpServer.Mux.Handler(),
		grpcClient,
		getDeprecatedLogStream,
		agent.GetTestWorkflowNotificationsStream(testWorkflowResultsRepository, executionWorker),
		agent.GetTestWorkflowServiceNotificationsStream(testWorkflowResultsRepository, executionWorker),
		agent.GetTestWorkflowParallelStepNotificationsStream(testWorkflowResultsRepository, executionWorker),
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
			deprecatedSystem,
			clientset,
			testkubeClientset,
			testWorkflowsClient,
			triggerLeaseBackend,
			log.DefaultLogger,
			configMapConfig,
			eventBus,
			metrics,
			executionWorker,
			testWorkflowExecutor,
			testWorkflowResultsRepository,
			triggers.WithHostnameIdentifier(),
			triggers.WithTestkubeNamespace(cfg.TestkubeNamespace),
			triggers.WithWatcherNamespaces(cfg.TestkubeWatcherNamespaces),
			triggers.WithDisableSecretCreation(!secretConfig.AutoCreate),
			triggers.WithProContext(&proContext),
		)
		log.DefaultLogger.Info("starting trigger service")
		g.Go(func() error {
			triggerService.Run(ctx)
			return nil
		})
	} else {
		log.DefaultLogger.Info("test triggers are disabled")
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

	if deprecatedSystem != nil {
		if deprecatedSystem.Reconciler != nil {
			g.Go(func() error {
				return deprecatedSystem.Reconciler.Run(ctx)
			})
		}

		if deprecatedSystem.API != nil {
			g.Go(func() error {
				return deprecatedSystem.API.RunGraphQLServer(ctx)
			})
		}
	}

	if err := g.Wait(); err != nil {
		log.DefaultLogger.Fatalf("Testkube is shutting down: %v", err)
	}
}
