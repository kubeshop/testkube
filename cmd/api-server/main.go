package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/zapr"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	k8sctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	testexecutionv1 "github.com/kubeshop/testkube-operator/api/testexecution/v1"
	testsuiteexecutionv1 "github.com/kubeshop/testkube-operator/api/testsuiteexecution/v1"
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	testkubeclientset "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	cronjobclient "github.com/kubeshop/testkube-operator/pkg/cronjob/client"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/cmd/api-server/services"
	"github.com/kubeshop/testkube/internal/app/api/debug"
	"github.com/kubeshop/testkube/internal/common"
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	cloudartifacts "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
	"github.com/kubeshop/testkube/pkg/controller"
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
	"github.com/kubeshop/testkube/pkg/scheduler"
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
	"github.com/kubeshop/testkube/pkg/controlplane"
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
	if cfg.TestkubeProAPIKey != "" || cfg.TestkubeProAgentRegToken != "" {
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
	var controlPlane *controlplane.Server
	if mode == common.ModeStandalone {
		controlPlane = services.CreateControlPlane(ctx, cfg, features, secretManager, metrics, lazyRunner, lazyEmitter)
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

	// If we don't have an API key but we do have a token for registration then attempt to register the runner.
	if cfg.TestkubeProAPIKey == "" && cfg.TestkubeProAgentRegToken != "" {
		runnerName := cfg.RunnerName
		if runnerName == "" {
			// Fallback to a set name, but this is unlikely to be unique.
			runnerName = cfg.APIServerFullname
		}
		log.DefaultLogger.Infow("registering runner", "runner_name", runnerName)

		// Check for required fields.
		if cfg.TestkubeProOrgID == "" {
			log.DefaultLogger.Fatalw("cannot register runner without org id", "error", "org id must be set to register a runner")
		}
		if cfg.SelfRegistrationSecret == "" {
			log.DefaultLogger.Fatalw("cannot register runner without self registration secret", "error", "self registration secret must be set to register a runner")
		}
		// If not configured to store secrets then registering the runner could cause severe issues such as
		// the runner registering on every restart creating new runner IDs in the Control Plane.
		if !(cfg.EnableSecretsEndpoint && !cfg.DisableSecretCreation) {
			log.DefaultLogger.Fatalw("cannot register runner without secrets enabled", "error", "secrets must be enabled to register a runner")
		}

		res, err := grpcClient.Register(ctx, &cloud.RegisterRequest{
			RegistrationToken: cfg.TestkubeProAgentRegToken,
			RunnerName:        runnerName,
			OrganizationId:    cfg.TestkubeProOrgID,
			Floating:          cfg.FloatingRunner,
		})
		if err != nil {
			log.DefaultLogger.Fatalw("error registering runner", "error", err.Error())
		}

		// Add the new values to the current configuration.
		cfg.TestkubeProAPIKey = res.RunnerKey
		cfg.TestkubeProAgentID = res.RunnerId

		// Attempt to store the values in a Kubernetes secret for consumption next startup.
		if _, err := secretManager.Create(ctx, cfg.TestkubeNamespace, cfg.SelfRegistrationSecret, map[string]string{
			"TESTKUBE_PRO_API_KEY":  res.RunnerKey,
			"TESTKUBE_PRO_AGENT_ID": res.RunnerId,
		}, secretmanager.CreateOptions{}); err != nil {
			log.DefaultLogger.Errorw("error creating self-register runner secret", "error", err.Error())
			log.DefaultLogger.Warn("runner will re-register on restart")
		}
	}

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

	// Create Kubernetes Operators/Controllers
	if cfg.EnableK8sControllers {
		// Initialise the controller runtime with our logger.
		ctrl.SetLogger(zapr.NewLogger(log.DefaultLogger.Desugar()))

		// Configure a scheme to include the required resource definitions.
		scheme := runtime.NewScheme()
		err = testworkflowsv1.AddToScheme(scheme)
		commons.ExitOnError("Add TestWorkflows to kubernetes runtime scheme", err)

		// Legacy schemes
		err = testexecutionv1.AddToScheme(scheme)
		commons.ExitOnError("Add TestExecution to kubernetes runtime scheme", err)
		err = testsuiteexecutionv1.AddToScheme(scheme)
		commons.ExitOnError("Add TestSuiteExecution to kubernetes runtime scheme", err)

		// Configure the manager to use the defined scheme and to operate in the current namespace.
		mgr, err := manager.New(kubeConfig, manager.Options{
			Scheme: scheme,
			Cache: cache.Options{
				DefaultNamespaces: map[string]cache.Config{
					cfg.TestkubeNamespace: {},
				},
			},
		})
		commons.ExitOnError("Creating kubernetes controller manager", err)

		// Initialise controllers
		err = controller.NewTestWorkflowExecutionExecutorController(mgr, testWorkflowExecutor)
		commons.ExitOnError("Creating TestWorkflowExecution controller", err)

		// Legacy controllers
		testExecutor := workerpool.New[testkube.Test, testkube.ExecutionRequest, testkube.Execution](scheduler.DefaultConcurrencyLevel)
		err = controller.NewTestExecutionExecutorController(mgr, testExecutor, deprecatedSystem)
		commons.ExitOnError("Creating TestExecution controller", err)
		testSuiteExecutor := workerpool.New[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution](scheduler.DefaultConcurrencyLevel)
		err = controller.NewTestSuiteExecutionExecutorController(mgr, testSuiteExecutor, deprecatedSystem)
		commons.ExitOnError("Creating TestSuiteExecution controller", err)

		// Finally start the manager.
		g.Go(func() error {
			return mgr.Start(ctx)
		})
	}

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

	if !cfg.DisableTestTriggers && controlPlane != nil {
		k8sCfg, err := k8sclient.GetK8sClientConfig()
		commons.ExitOnError("Getting k8s client config", err)
		testkubeClientset, err := testkubeclientset.NewForConfig(k8sCfg)
		commons.ExitOnError("Creating TestKube Clientset", err)
		// TODO: Check why this simpler options is not working
		//testkubeClientset := testkubeclientset.New(clientset.RESTClient())
		leaseBackend := controlPlane.GetRepositoryManager().LeaseBackend()
		triggerService := triggers.NewService(
			deprecatedSystem,
			clientset,
			testkubeClientset,
			testWorkflowsClient,
			leaseBackend,
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
		// Remove any remaining legacy cronjobs.
		// TODO: Remove this section once we are happy that users are not migrating from legacy cronjobs (next Major version?)
		resources := []string{cronjobclient.TestResourceURI, cronjobclient.TestSuiteResourceURI, cronjobclient.TestWorkflowResourceURI}
		for _, resource := range resources {
			reqs, err := labels.ParseToRequirements("testkube=" + resource)
			if err != nil {
				log.DefaultLogger.Errorw("Unable to parse label selector", "error", err, "label", "testkube="+resource)
				continue
			}

			u := &unstructured.Unstructured{}
			u.SetKind("CronJob")
			u.SetAPIVersion("batch/v1")
			if err := kubeClient.DeleteAllOf(
				ctx,
				u,
				k8sctrlclient.InNamespace(cfg.TestkubeNamespace),
				k8sctrlclient.MatchingLabelsSelector{Selector: labels.NewSelector().Add(reqs...)},
			); err != nil {
				log.DefaultLogger.Errorw("Unable to delete legacy cronjobs", "error", err, "label", "testkube="+resource, "namespace", cfg.TestkubeNamespace)
				continue
			}
		}

		// Start the new scheduler.
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
