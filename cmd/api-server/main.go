package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2/middleware/cors"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	testkubeclientset "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/cmd/api-server/services"
	"github.com/kubeshop/testkube/internal/app/api/debug"
	"github.com/kubeshop/testkube/internal/common"
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	cloudartifacts "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/crdstorage"
	"github.com/kubeshop/testkube/pkg/event/kind/cdevent"
	"github.com/kubeshop/testkube/pkg/event/kind/k8sevent"
	"github.com/kubeshop/testkube/pkg/event/kind/testworkflowexecutionmetrics"
	"github.com/kubeshop/testkube/pkg/event/kind/testworkflowexecutions"
	"github.com/kubeshop/testkube/pkg/event/kind/testworkflowexecutiontelemetry"
	"github.com/kubeshop/testkube/pkg/event/kind/webhook"
	ws "github.com/kubeshop/testkube/pkg/event/kind/websocket"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	runner2 "github.com/kubeshop/testkube/pkg/runner"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/kubeshop/testkube/pkg/tcl/schedulertcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/presets"
	"github.com/kubeshop/testkube/pkg/version"
	"github.com/kubeshop/testkube/pkg/workerpool"

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
	} else {
		cfg.TestkubeProURL = fmt.Sprintf("%s:%d", cfg.APIServerFullname, cfg.GRPCServerPort)
		cfg.TestkubeProTLSInsecure = true
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

	// k8s
	kubeClient, err := kubeclient.GetClient()
	commons.ExitOnError("Getting kubernetes client", err)
	kubeConfig, err := k8sclient.GetK8sClientConfig()
	commons.ExitOnError("Getting kubernetes config", err)
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	commons.ExitOnError("Creating k8s clientset", err)

	var eventsEmitter *event.Emitter
	lazyEmitter := event.Lazy(&eventsEmitter)

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

	metrics := metrics.NewMetrics()

	lazyRunner := runner2.LazyExecute()

	// Connect to the Control Plane
	var grpcConn *grpc.ClientConn
	if mode == common.ModeStandalone {
		controlPlane := services.CreateControlPlane(ctx, cfg, features, secretManager, metrics, lazyRunner, lazyEmitter)
		g.Go(func() error {
			return controlPlane.Start(ctx)
		})
		grpcConn, err = agentclient.NewGRPCConnection(ctx, true, true, fmt.Sprintf("127.0.0.1:%d", cfg.GRPCServerPort), "", "", "", log.DefaultLogger)
	} else {
		grpcConn, err = agentclient.NewGRPCConnection(
			ctx,
			cfg.TestkubeProTLSInsecure,
			cfg.TestkubeProSkipVerify,
			cfg.TestkubeProURL,
			cfg.TestkubeProCertFile,
			cfg.TestkubeProKeyFile,
			cfg.TestkubeProCAFile, //nolint
			log.DefaultLogger,
		)
	}
	commons.ExitOnError("error creating gRPC connection", err)
	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)

	clusterId, _ := configMapConfig.GetUniqueClusterId(ctx)
	telemetryEnabled, _ := configMapConfig.GetTelemetryEnabled(ctx)

	// k8s clients
	webhooksClient := executorsclientv1.NewWebhooksClient(kubeClient, cfg.TestkubeNamespace)
	webhookTemplatesClient := executorsclientv1.NewWebhookTemplatesClient(kubeClient, cfg.TestkubeNamespace)
	testTriggersClient := testtriggersclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)

	envs := commons.GetEnvironmentVariables()

	inspector := commons.CreateImageInspector(&cfg.ImageInspectorConfig, configmap.NewClientFor(clientset, cfg.TestkubeNamespace), secret.NewClientFor(clientset, cfg.TestkubeNamespace))

	var testWorkflowsClient testworkflowclient.TestWorkflowClient
	var testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient

	testWorkflowResultsRepository := cloudtestworkflow.NewCloudRepository(grpcClient, cfg.TestkubeProAPIKey)
	testWorkflowOutputRepository := cloudtestworkflow.NewCloudOutputRepository(grpcClient, cfg.TestkubeProAPIKey, cfg.StorageSkipVerify)
	webhookRepository := cloudwebhook.NewCloudRepository(grpcClient, cfg.TestkubeProAPIKey)
	triggerLeaseBackend := triggers.NewAcquireAlwaysLeaseBackend()
	artifactStorage := cloudartifacts.NewCloudArtifactsStorage(grpcClient, cfg.TestkubeProAPIKey)

	nc := commons.MustCreateNATSConnection(cfg)
	eventBus := bus.NewNATSBus(nc)
	if cfg.Trace {
		eventBus.TraceEvents()
	}
	eventsEmitter = event.NewEmitter(eventBus, cfg.TestkubeClusterName)

	proContext := commons.ReadProContext(ctx, cfg, grpcClient)

	// Build new client
	client := controlplaneclient.New(grpcClient, proContext, controlplaneclient.ClientOptions{
		StorageSkipVerify: cfg.StorageSkipVerify,
		Runtime: controlplaneclient.RuntimeConfig{
			Namespace: cfg.TestkubeNamespace,
		},
		SendTimeout: cfg.TestkubeProSendTimeout,
		RecvTimeout: cfg.TestkubeProRecvTimeout,
	})

	if proContext.CloudStorage {
		testWorkflowsClient = testworkflowclient.NewCloudTestWorkflowClient(client)
		testWorkflowTemplatesClient = testworkflowtemplateclient.NewCloudTestWorkflowTemplateClient(client)
	} else {
		testWorkflowsClient, err = testworkflowclient.NewKubernetesTestWorkflowClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)
		commons.ExitOnError("Creating test workflow client", err)
		testWorkflowTemplatesClient, err = testworkflowtemplateclient.NewKubernetesTestWorkflowTemplateClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)
		commons.ExitOnError("Creating test workflow templates client", err)
	}

	defaultExecutionNamespace := cfg.TestkubeNamespace
	if cfg.DefaultExecutionNamespace != "" {
		defaultExecutionNamespace = cfg.DefaultExecutionNamespace
	}
	serviceAccountNames := map[string]string{
		defaultExecutionNamespace: cfg.JobServiceAccountName,
	}
	// Pro edition only (tcl protected code)
	if cfg.TestkubeExecutionNamespaces != "" {
		if mode != common.ModeAgent {
			commons.ExitOnError("Execution namespaces", common.ErrNotSupported)
		}

		serviceAccountNames = schedulertcl.GetServiceAccountNamesFromConfig(serviceAccountNames, cfg.TestkubeExecutionNamespaces)
	}

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
			nc,
			eventsEmitter,
			eventBus,
			inspector,
			&proContext,
		)
	}

	// Transfer common environment variables
	commonEnvVariables := make([]corev1.EnvVar, 0)
	for _, envName := range cfg.TransferEnvVariables {
		if value := os.Getenv(envName); value != "" {
			commonEnvVariables = append(commonEnvVariables, corev1.EnvVar{Name: envName, Value: value})
		}
	}

	// Build internal execution worker
	testWorkflowProcessor := presets.NewOpenSource(inspector)
	// Pro edition only (tcl protected code)
	if mode == common.ModeAgent {
		testWorkflowProcessor = presets.NewPro(inspector)
	}
	executionWorker := services.CreateExecutionWorker(clientset, cfg, clusterId, proContext.Agent.ID, serviceAccountNames, testWorkflowProcessor, map[string]string{
		testworkflowconfig.FeatureFlagNewArchitecture: fmt.Sprintf("%v", cfg.FeatureNewArchitecture),
		testworkflowconfig.FeatureFlagCloudStorage:    fmt.Sprintf("%v", cfg.FeatureCloudStorage),
	}, commonEnvVariables, true, defaultExecutionNamespace)

	runnerOpts := runner2.Options{
		ClusterID:           clusterId,
		DefaultNamespace:    defaultExecutionNamespace,
		ServiceAccountNames: serviceAccountNames,
		StorageSkipVerify:   cfg.StorageSkipVerify,
	}
	if cfg.GlobalWorkflowTemplateInline != "" {
		runnerOpts.GlobalTemplate = runner2.GlobalTemplateInline(cfg.GlobalWorkflowTemplateInline)
	} else if cfg.GlobalWorkflowTemplateName != "" && cfg.FeatureNewArchitecture && proContext.NewArchitecture {
		runnerOpts.GlobalTemplate = runner2.GlobalTemplateSourced(testWorkflowTemplatesClient, cfg.GlobalWorkflowTemplateName)
	}
	runnerService := runner2.NewService(
		log.DefaultLogger,
		eventsEmitter,
		metrics,
		configMapConfig,
		client,
		testworkflowconfig.ControlPlaneConfig{
			DashboardUrl:   proContext.DashboardURI,
			CDEventsTarget: cfg.CDEventsTarget,
		},
		proContext,
		executionWorker,
		runnerOpts,
	)
	if !cfg.DisableRunner {
		g.Go(func() error {
			return runnerService.Start(ctx)
		})
	}
	lazyRunner.Set(runnerService)

	testWorkflowExecutor := testworkflowexecutor.New(
		grpcClient,
		cfg.TestkubeProAPIKey,
		cfg.CDEventsTarget,
		eventsEmitter,
		runnerService,
		testWorkflowResultsRepository,
		testWorkflowOutputRepository,
		testWorkflowTemplatesClient,
		testWorkflowsClient,
		metrics,
		secretManager,
		cfg.GlobalWorkflowTemplateName,
		proContext.DashboardURI,
		proContext.OrgID,
		proContext.OrgSlug,
		proContext.EnvID,
		proContext.GetEnvSlug,
		proContext.Agent.ID,
		proContext.NewArchitecture,
	)

	var deprecatedClients commons.DeprecatedClients
	var deprecatedRepositories commons.DeprecatedRepositories
	if deprecatedSystem != nil {
		deprecatedClients = deprecatedSystem.Clients
		deprecatedRepositories = deprecatedSystem.Repositories
	}

	// Initialize event handlers
	websocketLoader := ws.NewWebsocketLoader()
	if !cfg.DisableWebhooks {
		secretClient := secret.NewClientFor(clientset, cfg.TestkubeNamespace)
		eventsEmitter.Loader.Register(webhook.NewWebhookLoader(log.DefaultLogger, webhooksClient, webhookTemplatesClient, deprecatedClients, deprecatedRepositories,
			testWorkflowResultsRepository, secretClient, metrics, webhookRepository, &proContext, envs))
	}
	eventsEmitter.Loader.Register(websocketLoader)
	eventsEmitter.Loader.Register(commons.MustCreateSlackLoader(cfg, envs))
	if cfg.CDEventsTarget != "" {
		cdeventLoader, err := cdevent.NewCDEventLoader(cfg.CDEventsTarget, clusterId, cfg.TestkubeNamespace, proContext.DashboardURI, testkube.AllEventTypes)
		if err == nil {
			eventsEmitter.Loader.Register(cdeventLoader)
		} else {
			log.DefaultLogger.Debugw("cdevents init error", "error", err.Error())
		}
	}
	if cfg.EnableK8sEvents {
		eventsEmitter.Loader.Register(k8sevent.NewK8sEventLoader(clientset, cfg.TestkubeNamespace, testkube.AllEventTypes))
	}

	// Update the Prometheus metrics regarding the Test Workflow Execution
	eventsEmitter.Loader.Register(testworkflowexecutionmetrics.NewLoader(ctx, metrics, proContext.DashboardURI))

	// Send the telemetry data regarding the Test Workflow Execution
	// TODO: Disable it if Control Plane does that
	eventsEmitter.Loader.Register(testworkflowexecutiontelemetry.NewLoader(ctx, configMapConfig))

	// Update TestWorkflowExecution Kubernetes resource objects on status change
	eventsEmitter.Loader.Register(testworkflowexecutions.NewLoader(ctx, cfg.TestkubeNamespace, kubeClient))

	// Synchronise Test Workflows with cloud
	if proContext.CloudStorageSupportedInControlPlane && (cfg.GitOpsSyncKubernetesToCloudEnabled || cfg.GitOpsSyncCloudToKubernetesEnabled) {
		testWorkflowsCloudStorage, err := crdstorage.NewTestWorkflowsStorage(testworkflowclient.NewCloudTestWorkflowClient(client), proContext.EnvID, cfg.GitOpsSyncCloudNamePattern, nil)
		commons.ExitOnError("connecting to cloud TestWorkflows storage", err)
		testWorkflowsKubernetesStorage, err := crdstorage.NewTestWorkflowsStorage(must(testworkflowclient.NewKubernetesTestWorkflowClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)), proContext.EnvID, cfg.GitOpsSyncKubernetesNamePattern, map[string]string{
			"namespace": cfg.TestkubeNamespace,
		})
		commons.ExitOnError("connecting to k8s TestWorkflows storage", err)
		testWorkflowTemplatesCloudStorage, err := crdstorage.NewTestWorkflowTemplatesStorage(testworkflowtemplateclient.NewCloudTestWorkflowTemplateClient(client), proContext.EnvID, cfg.GitOpsSyncCloudNamePattern, nil)
		commons.ExitOnError("connecting to cloud TestWorkflowTemplates storage", err)
		testWorkflowTemplatesKubernetesStorage, err := crdstorage.NewTestWorkflowTemplatesStorage(must(testworkflowtemplateclient.NewKubernetesTestWorkflowTemplateClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)), proContext.EnvID, cfg.GitOpsSyncKubernetesNamePattern, map[string]string{
			"namespace": cfg.TestkubeNamespace,
		})
		commons.ExitOnError("connecting to k8s TestWorkflowTemplates storage", err)

		if cfg.GitOpsSyncCloudToKubernetesEnabled && cfg.FeatureCloudStorage {
			// Test Workflows - Continuous Sync (eventual) - Cloud -> Kubernetes
			g.Go(func() error {
				for {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					watcher := testWorkflowsCloudStorage.Watch(ctx)
					for obj := range watcher.Channel() {
						err := testWorkflowsKubernetesStorage.Process(ctx, obj)
						if err == nil {
							log.DefaultLogger.Infow("synced TestWorkflow from Control Plane in Kubernetes", "name", obj.Resource.Name, "error", err)
						} else {
							log.DefaultLogger.Errorw("failed to include TestWorkflow in Kubernetes", "error", err)
						}
					}
					if watcher.Err() != nil {
						log.DefaultLogger.Errorw("failed to watch TestWorkflows in Kubernetes", "error", watcher.Err())
					}

					time.Sleep(200 * time.Millisecond)
				}
			})

			// Test Workflow Templates - Continuous Sync (eventual) - Cloud -> Kubernetes
			g.Go(func() error {
				for {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					watcher := testWorkflowTemplatesCloudStorage.Watch(ctx)
					for obj := range watcher.Channel() {
						err := testWorkflowTemplatesKubernetesStorage.Process(ctx, obj)
						if err == nil {
							log.DefaultLogger.Infow("synced TestWorkflowTemplate from Control Plane in Kubernetes", "name", obj.Resource.Name, "error", err)
						} else {
							log.DefaultLogger.Errorw("failed to include TestWorkflowTemplate in Kubernetes", "error", err)
						}
					}
					if watcher.Err() != nil {
						log.DefaultLogger.Errorw("failed to watch TestWorkflowTemplates in Control Plane", "error", watcher.Err())
					}

					time.Sleep(200 * time.Millisecond)
				}
			})
		}

		if cfg.GitOpsSyncKubernetesToCloudEnabled {
			// Test Workflows - Continuous Sync (eventual) - Kubernetes -> Cloud
			g.Go(func() error {
				for {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					watcher := testWorkflowsKubernetesStorage.Watch(ctx)
					for obj := range watcher.Channel() {
						err := testWorkflowsCloudStorage.Process(ctx, obj)
						if err == nil {
							log.DefaultLogger.Infow("synced TestWorkflow from Kubernetes into Control Plane", "name", obj.Resource.Name, "error", err)
						} else {
							log.DefaultLogger.Errorw("failed to include TestWorkflow in Control Plane", "error", err)
						}
					}
					if watcher.Err() != nil {
						log.DefaultLogger.Errorw("failed to watch TestWorkflows in Kubernetes", "error", watcher.Err())
					}

					time.Sleep(200 * time.Millisecond)
				}
			})

			// Test Workflow Templates - Continuous Sync (eventual) - Kubernetes -> Cloud
			g.Go(func() error {
				for {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					watcher := testWorkflowTemplatesKubernetesStorage.Watch(ctx)
					for obj := range watcher.Channel() {
						err := testWorkflowTemplatesCloudStorage.Process(ctx, obj)
						if err == nil {
							log.DefaultLogger.Infow("synced TestWorkflowTemplate from Kubernetes into Control Plane", "name", obj.Resource.Name, "error", err)
						} else {
							log.DefaultLogger.Errorw("failed to include TestWorkflowTemplate in Control Plane", "error", err)
						}
					}
					if watcher.Err() != nil {
						log.DefaultLogger.Errorw("failed to watch TestWorkflowTemplates in Kubernetes", "error", watcher.Err())
					}

					time.Sleep(200 * time.Millisecond)
				}
			})
		}
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
		webhookTemplatesClient,
		testTriggersClient,
		testWorkflowsClient,
		testworkflowsclientv1.NewClient(kubeClient, cfg.TestkubeNamespace),
		testWorkflowTemplatesClient,
		testworkflowsclientv1.NewTestWorkflowTemplatesClient(kubeClient, cfg.TestkubeNamespace),
		configMapConfig,
		secretManager,
		secretConfig,
		executionWorker,
		eventsEmitter,
		websocketLoader,
		metrics,
		&proContext,
		features,
		cfg.TestkubeHelmchartVersion,
		serviceAccountNames,
		cfg.TestkubeDockerImageVersion,
		testWorkflowExecutor,
	)
	api.Init(httpServer)

	log.DefaultLogger.Info("starting agent service")

	getDeprecatedLogStream := agent.GetDeprecatedLogStream
	if deprecatedSystem != nil && deprecatedSystem.StreamLogs != nil {
		getDeprecatedLogStream = deprecatedSystem.StreamLogs
	}
	if !cfg.DisableDefaultAgent {
		agentHandle, err := agent.NewAgent(
			log.DefaultLogger,
			httpServer.Mux.Handler(),
			grpcClient,
			getDeprecatedLogStream,
			clusterId,
			cfg.TestkubeClusterName,
			features,
			&proContext,
			cfg.TestkubeDockerImageVersion,
			eventsEmitter,
		)
		commons.ExitOnError("Starting agent", err)
		g.Go(func() error {
			err = agentHandle.Run(ctx)
			commons.ExitOnError("Running agent", err)
			return nil
		})
		eventsEmitter.Loader.Register(agentHandle)
	}

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
		"executionNamespace", cfg.DefaultExecutionNamespace,
		"version", version.Version,
	)

	if cfg.EnableDebugServer {
		debugSrv := debug.NewDebugServer(cfg.DebugListenAddr)

		g.Go(func() error {
			log.DefaultLogger.Infof("starting debug pprof server")
			return debugSrv.ListenAndServe()
		})
	}

	var executeTestFn workerpool.ExecuteFn[testkube.Test, testkube.ExecutionRequest, testkube.Execution]
	var executeTestSuiteFn workerpool.ExecuteFn[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution]
	if deprecatedSystem != nil && deprecatedSystem.Scheduler != nil {
		executeTestFn = deprecatedSystem.Scheduler.ExecuteTest
		executeTestSuiteFn = deprecatedSystem.Scheduler.ExecuteTestSuite
	}

	scheduler := commons.CreateCronJobScheduler(cfg,
		kubeClient,
		testWorkflowsClient,
		testWorkflowTemplatesClient,
		testWorkflowExecutor,
		deprecatedClients,
		executeTestFn,
		executeTestSuiteFn,
		log.DefaultLogger,
		kubeConfig,
		&proContext,
	)
	if scheduler != nil {
		g.Go(func() error {
			scheduler.Reconcile(ctx)
			return nil
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

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
