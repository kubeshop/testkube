package main

import (
	"context"
	"flag"
	"fmt"
	"net"
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

	testexecutionv1 "github.com/kubeshop/testkube/api/testexecution/v1"
	testsuiteexecutionv1 "github.com/kubeshop/testkube/api/testsuiteexecution/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
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
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/crdstorage"
	"github.com/kubeshop/testkube/pkg/event/kind/cdevent"
	"github.com/kubeshop/testkube/pkg/event/kind/k8sevent"
	"github.com/kubeshop/testkube/pkg/event/kind/testworkflowexecutionmetrics"
	"github.com/kubeshop/testkube/pkg/event/kind/testworkflowexecutions"
	"github.com/kubeshop/testkube/pkg/event/kind/testworkflowexecutiontelemetry"
	"github.com/kubeshop/testkube/pkg/event/kind/webhook"
	ws "github.com/kubeshop/testkube/pkg/event/kind/websocket"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/newclients/webhookclient"
	observtracing "github.com/kubeshop/testkube/pkg/observability/tracing"
	executorsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/executors/v1"
	testkubeclientset "github.com/kubeshop/testkube/pkg/operator/clientset/versioned"
	cronjobclient "github.com/kubeshop/testkube/pkg/operator/cronjob/client"
	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
	leasebackendk8s "github.com/kubeshop/testkube/pkg/repository/leasebackend/k8s"
	runner2 "github.com/kubeshop/testkube/pkg/runner"
	runnergrpc "github.com/kubeshop/testkube/pkg/runner/grpc"
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

	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/controlplane"
	"github.com/kubeshop/testkube/pkg/log"
	kubeclient "github.com/kubeshop/testkube/pkg/operator/client"
	testtriggersclientv1 "github.com/kubeshop/testkube/pkg/operator/client/testtriggers/v1"
	testworkflowsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

func init() {
	flag.Parse()
}

func main() {
	startTime := time.Now()
	log.DefaultLogger.Info("starting Testkube API Server")
	log.DefaultLogger.Infow("version info", "version", version.Version, "commit", version.Commit)

	cfg := commons.MustGetConfig()
	features := commons.MustGetFeatureFlags()

	mode := common.ModeStandalone
	if cfg.TestkubeProAPIKey != "" || cfg.TestkubeProAgentRegToken != "" {
		mode = common.ModeAgent
	} else {
		cfg.TestkubeProURL = fmt.Sprintf("%s:%d", cfg.APIServerFullname, cfg.GRPCServerPort)
		cfg.TestkubeProTLSInsecure = true
	}

	log.DefaultLogger.Infow("configuration loaded",
		"mode", mode,
		"namespace", cfg.TestkubeNamespace,
		"apiServerPort", cfg.APIServerPort,
		"grpcPort", cfg.GRPCServerPort,
	)

	shutdownTracing, err := observtracing.Init(context.Background(), observtracing.Config{
		Enabled:       cfg.TracingEnabled,
		Endpoint:      cfg.OTLPEndpoint,
		ServiceName:   cfg.OTLPServiceName,
		SamplingRatio: cfg.TracingSampleRate,
		Version:       version.Version,
		Commit:        version.Commit,
	})
	commons.ExitOnError("initializing tracing", err)
	defer func() { _ = shutdownTracing(context.Background()) }()

	// Determine the running mode

	// Run services within an errgroup to propagate errors between services.
	g, ctx := errgroup.WithContext(context.Background())

	// Cancel the errgroup context on SIGINT and SIGTERM,
	// which shuts everything down gracefully.
	g.Go(commons.HandleCancelSignal(ctx))

	commons.MustFreePort(cfg.APIServerPort)
	commons.MustFreePort(cfg.GRPCServerPort)

	log.DefaultLogger.Info("initializing...")
	configMapConfig := commons.MustGetConfigMapConfig(ctx, cfg.APIServerConfig, cfg.TestkubeNamespace, cfg.TestkubeAnalyticsEnabled)
	log.DefaultLogger.Info("ConfigMap configuration loaded successfully")

	// k8s
	log.DefaultLogger.Info("connecting to Kubernetes cluster...")
	kubeClient, err := kubeclient.GetClient()
	commons.ExitOnError("getting Kubernetes client", err)
	kubeConfig, err := k8sclient.GetK8sClientConfig()
	commons.ExitOnError("getting Kubernetes config", err)
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	commons.ExitOnError("creating k8s clientset", err)

	log.DefaultLogger.Infow("connected to Kubernetes cluster successfully", "namespace", cfg.TestkubeNamespace)

	var eventsEmitter *event.Emitter

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

	// Connect to the Control Plane
	var grpcConn *grpc.ClientConn
	var controlPlane *controlplane.Server
	if mode == common.ModeStandalone {
		log.DefaultLogger.Info("starting embedded Control Plane service...")
		controlPlane = services.CreateControlPlane(ctx, cfg, features)

		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCServerPort))
		commons.ExitOnError("cannot listen to gRPC port", err)
		g.Go(func() error {
			return controlPlane.Start(ctx, ln)
		})
		grpcConn, err = agentclient.NewGRPCConnectionWithTracing(ctx, true, true, fmt.Sprintf("127.0.0.1:%d", cfg.GRPCServerPort), "", log.DefaultLogger, cfg.TracingEnabled)
		commons.ExitOnError("connecting to embedded Control Plane", err)
		log.DefaultLogger.Infow("connected to embedded control plane successfully", "port", cfg.GRPCServerPort)
	} else {
		log.DefaultLogger.Infow("connecting to remote control plane...", "url", cfg.TestkubeProURL)
		var err error
		grpcConn, err = agentclient.NewGRPCConnectionWithTracing(
			ctx,
			cfg.TestkubeProTLSInsecure,
			cfg.TestkubeProSkipVerify,
			cfg.TestkubeProURL,
			cfg.TestkubeProCAFile, //nolint
			log.DefaultLogger,
			cfg.TracingEnabled,
		)
		commons.ExitOnError("connecting to remote Control Plane", err)
		log.DefaultLogger.Infow("connected to remote control plane successfully", "url", cfg.TestkubeProURL)
	}
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
		if !cfg.EnableSecretsEndpoint || cfg.DisableSecretCreation {
			log.DefaultLogger.Fatalw("cannot register runner without secrets enabled", "error", "secrets must be enabled to register a runner")
		}

		// Build capabilities based on enabled features
		capabilities := []cloud.AgentCapability{}
		if !cfg.DisableRunner {
			capabilities = append(capabilities, cloud.AgentCapability_AGENT_CAPABILITY_RUNNER)
		}
		if !cfg.DisableTestTriggers {
			capabilities = append(capabilities, cloud.AgentCapability_AGENT_CAPABILITY_LISTENER)
		}

		res, err := grpcClient.Register(ctx, &cloud.RegisterRequest{
			RegistrationToken: cfg.TestkubeProAgentRegToken,
			RunnerName:        runnerName,
			OrganizationId:    cfg.TestkubeProOrgID,
			Floating:          cfg.FloatingRunner,
			Capabilities:      capabilities,
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

	envs := commons.GetEnvironmentVariables()

	inspector := commons.CreateImageInspector(&cfg.ImageInspectorConfig, configmap.NewClientFor(clientset, cfg.TestkubeNamespace), secret.NewClientFor(clientset, cfg.TestkubeNamespace))

	var (
		testWorkflowsClient         testworkflowclient.TestWorkflowClient
		testWorkflowTemplatesClient testworkflowtemplateclient.TestWorkflowTemplateClient
		testTriggersClient          testtriggerclient.TestTriggerClient
	)
	proContext, err := commons.ReadProContext(ctx, cfg, grpcClient)
	commons.ExitOnError("cannot connect to control plane", err)

	testWorkflowResultsRepository := cloudtestworkflow.NewCloudRepository(grpcClient, &proContext)
	testWorkflowOutputRepository := cloudtestworkflow.NewCloudOutputRepository(grpcClient, cfg.StorageSkipVerify, &proContext)
	webhookRepository := cloudwebhook.NewCloudRepository(grpcClient, &proContext)
	artifactStorage := cloudartifacts.NewCloudArtifactsStorage(grpcClient, &proContext)

	log.DefaultLogger.Info("connecting to NATS...")
	nc := commons.MustCreateNATSConnection(cfg)
	log.DefaultLogger.Infow("connected to NATS successfully", "embedded", cfg.NatsEmbedded, "uri", cfg.NatsURI)

	eventBus := bus.NewNATSBus(nc)
	if cfg.Trace {
		eventBus.TraceEvents()
	}
	eventsEmitter = event.NewEmitter(eventBus, cfg.TestkubeClusterName)

	// Build new client
	client := controlplaneclient.New(grpcClient, proContext, controlplaneclient.ClientOptions{
		StorageSkipVerify: cfg.StorageSkipVerify,
		Runtime: controlplaneclient.RuntimeConfig{
			Namespace: cfg.TestkubeNamespace,
		},
		SendTimeout: cfg.TestkubeProSendTimeout,
		RecvTimeout: cfg.TestkubeProRecvTimeout,
	}, log.DefaultLogger)

	if proContext.CloudStorage {
		testWorkflowsClient = testworkflowclient.NewCloudTestWorkflowClient(client)
		testWorkflowTemplatesClient = testworkflowtemplateclient.NewCloudTestWorkflowTemplateClient(client)
		testTriggersClient = testtriggerclient.NewCloudTestTriggerClient(client)
	} else {
		testWorkflowsClient, err = testworkflowclient.NewKubernetesTestWorkflowClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)
		commons.ExitOnError("creating test workflow client", err)
		testWorkflowTemplatesClient, err = testworkflowtemplateclient.NewKubernetesTestWorkflowTemplateClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)
		commons.ExitOnError("creating test workflow templates client", err)

		legacyTestTriggersClientForAPI := testtriggersclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)
		testTriggersClient = testtriggerclient.NewKubernetesTestTriggerClient(legacyTestTriggersClientForAPI)
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
			commons.ExitOnError("execution namespaces", common.ErrNotSupported)
		}

		serviceAccountNames = schedulertcl.GetServiceAccountNamesFromConfig(serviceAccountNames, cfg.TestkubeExecutionNamespaces)
	}

	var deprecatedSystem *services.DeprecatedSystem
	if !cfg.DisableDeprecatedTests {
		log.DefaultLogger.Info("initializing deprecated test system...")
		log.DefaultLogger.Info("  - connecting to MongoDB and other storage backends...")
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
		log.DefaultLogger.Info("deprecated test system initialized successfully")
	} else {
		log.DefaultLogger.Info("deprecated test system is disabled")
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
		testworkflowconfig.FeatureFlagCloudStorage: fmt.Sprintf("%v", cfg.FeatureCloudStorage),
	}, commonEnvVariables, true, defaultExecutionNamespace)

	runnerOpts := runner2.Options{
		ClusterID:           clusterId,
		DefaultNamespace:    defaultExecutionNamespace,
		ServiceAccountNames: serviceAccountNames,
		StorageSkipVerify:   cfg.StorageSkipVerify,
	}
	if cfg.GlobalWorkflowTemplateInline != "" {
		runnerOpts.GlobalTemplate = runner2.GlobalTemplateInline(cfg.GlobalWorkflowTemplateInline)
	} else if cfg.GlobalWorkflowTemplateName != "" {
		runnerOpts.GlobalTemplate = runner2.GlobalTemplateSourced(testWorkflowTemplatesClient, cfg.GlobalWorkflowTemplateName)
	}
	runner := runner2.New(
		executionWorker,
		configMapConfig,
		client,
		eventsEmitter,
		metrics,
		proContext,
		runnerOpts.StorageSkipVerify,
		runnerOpts.GlobalTemplate,
	)
	runnerService := runner2.NewService(
		log.DefaultLogger,
		eventsEmitter,
		client,
		testworkflowconfig.ControlPlaneConfig{
			DashboardUrl:   proContext.DashboardURI,
			CDEventsTarget: cfg.CDEventsTarget,
		},
		proContext,
		executionWorker,
		runnerOpts,
		runner,
	)

	runnerClient := runnergrpc.NewClient(
		grpcConn,
		log.DefaultLogger,
		runner,
		proContext.APIKey,
		proContext.OrgID,
		testworkflowconfig.ControlPlaneConfig{
			DashboardUrl:   proContext.DashboardURI,
			CDEventsTarget: cfg.CDEventsTarget,
		},
		testWorkflowsClient,
	)

	if !cfg.DisableRunner {
		g.Go(func() error {
			// Check if the new client is supported by the control plane.
			// If not then just start up the existing implementation.
			// Here we are using a context with a timeout because the client and/or server may not have TLS implemented as it was
			// not previously enforced, however it is required with the new client implementation to secure authentication tokens.
			// gRPC does not provide a specific error for an attempt to transmit credentials over an unencrypted connection so to
			// prevent the fallback to the previous insecure behaviour we must instead wait to see whether connectivity can be
			// established. The repercussions of this is that the agent will eagerly fallback to the insecure and legacy behaviour
			// and so bringing up an agent before networking with the Control Plane has been established, or during a Control Plane
			// outage will cause a fallback to the previous behaviour.
			// This timeout should be removed once TLS is enforced across deployments.
			supportedCtx, cancel := context.WithTimeout(ctx, time.Minute)
			if !runnerClient.IsSupported(supportedCtx, proContext.EnvID) {
				cancel()
				log.DefaultLogger.Warn("new runner RPC is not supported by Control Plane, falling back to previous implementation.")
				return runnerService.Start(ctx, true)
			}
			cancel()
			log.DefaultLogger.Info("new runner RPC is supported by Control Plane, will use new runner RPC to retrieve execution updates.")
			// If the client is supported then start both services/clients.
			var eg errgroup.Group
			eg.Go(func() error {
				// Start the older service but without the legacy execution RPC loop.
				return runnerService.Start(ctx, false)
			})
			eg.Go(func() error {
				return runnerClient.Start(ctx, proContext.EnvID)
			})
			return eg.Wait()
		})
	}

	testWorkflowExecutor := testworkflowexecutor.New(
		grpcClient,
		cfg.TestkubeProAPIKey,
		eventsEmitter,
		runnerService,
		proContext.OrgID,
		proContext.EnvID,
		proContext.Agent.ID,
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

	// Synchronise resources with cloud
	if proContext.CloudStorageSupportedInControlPlane && (cfg.GitOpsSyncKubernetesToCloudEnabled || cfg.GitOpsSyncCloudToKubernetesEnabled) {
		// TestWorkflows storage
		testWorkflowsCloudStorage, err := crdstorage.NewTestWorkflowsStorage(testworkflowclient.NewCloudTestWorkflowClient(client), proContext.EnvID, cfg.GitOpsSyncCloudNamePattern, nil)
		commons.ExitOnError("connecting to cloud TestWorkflows storage", err)
		testWorkflowsKubernetesStorage, err := crdstorage.NewTestWorkflowsStorage(must(testworkflowclient.NewKubernetesTestWorkflowClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)), proContext.EnvID, cfg.GitOpsSyncKubernetesNamePattern, map[string]string{
			"namespace": cfg.TestkubeNamespace,
		})
		commons.ExitOnError("connecting to k8s TestWorkflows storage", err)
		// TestWorkflowTemplates storage
		testWorkflowTemplatesCloudStorage, err := crdstorage.NewTestWorkflowTemplatesStorage(testworkflowtemplateclient.NewCloudTestWorkflowTemplateClient(client), proContext.EnvID, cfg.GitOpsSyncCloudNamePattern, nil)
		commons.ExitOnError("connecting to cloud TestWorkflowTemplates storage", err)
		testWorkflowTemplatesKubernetesStorage, err := crdstorage.NewTestWorkflowTemplatesStorage(must(testworkflowtemplateclient.NewKubernetesTestWorkflowTemplateClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)), proContext.EnvID, cfg.GitOpsSyncKubernetesNamePattern, map[string]string{
			"namespace": cfg.TestkubeNamespace,
		})
		commons.ExitOnError("connecting to k8s TestWorkflowTemplates storage", err)
		// Webhooks storage
		webhooksCloudStorage, err := crdstorage.NewWebhooksStorage(webhookclient.NewCloudWebhookClient(client), proContext.EnvID, cfg.GitOpsSyncCloudNamePattern, nil)
		commons.ExitOnError("connecting to cloud Webhooks storage", err)
		webhooksKubernetesStorage, err := crdstorage.NewWebhooksStorage(must(webhookclient.NewKubernetesWebhookClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)), proContext.EnvID, cfg.GitOpsSyncKubernetesNamePattern, map[string]string{
			"namespace": cfg.TestkubeNamespace,
		})
		commons.ExitOnError("connecting to k8s Webhooks storage", err)

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
							log.DefaultLogger.Infow("synced TestWorkflow from Control Plane in Kubernetes", "name", obj.Resource.Name)
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
							log.DefaultLogger.Infow("synced TestWorkflowTemplate from Control Plane in Kubernetes", "name", obj.Resource.Name)
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

			// Webhooks - Continuous Sync (eventual) - Cloud -> Kubernetes
			g.Go(func() error {
				for {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					watcher := webhooksCloudStorage.Watch(ctx)
					for obj := range watcher.Channel() {
						err := webhooksKubernetesStorage.Process(ctx, obj)
						if err == nil {
							log.DefaultLogger.Infow("synced Webhook from Control Plane in Kubernetes", "name", obj.Resource.Name)
						} else {
							log.DefaultLogger.Errorw("failed to include Webhook in Kubernetes", "error", err)
						}
					}
					if watcher.Err() != nil {
						log.DefaultLogger.Errorw("failed to watch Webhooks in Control Plane", "error", watcher.Err())
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
							log.DefaultLogger.Infow("synced TestWorkflow from Kubernetes into Control Plane", "name", obj.Resource.Name)
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
							log.DefaultLogger.Infow("synced TestWorkflowTemplate from Kubernetes into Control Plane", "name", obj.Resource.Name)
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

			// Webhooks - Continuous Sync (eventual) - Kubernetes -> Cloud
			g.Go(func() error {
				for {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					watcher := webhooksKubernetesStorage.Watch(ctx)
					for obj := range watcher.Channel() {
						err := webhooksCloudStorage.Process(ctx, obj)
						if err == nil {
							log.DefaultLogger.Infow("synced Webhook from Kubernetes into Control Plane", "name", obj.Resource.Name)
						} else {
							log.DefaultLogger.Errorw("failed to include Webhook in Control Plane", "error", err)
						}
					}
					if watcher.Err() != nil {
						log.DefaultLogger.Errorw("failed to watch Webhooks in Kubernetes", "error", watcher.Err())
					}

					time.Sleep(200 * time.Millisecond)
				}
			})
		}
	}

	log.DefaultLogger.Info("starting event system...")
	eventsEmitter.Listen(ctx)
	g.Go(func() error {
		eventsEmitter.Reconcile(ctx)
		return nil
	})
	log.DefaultLogger.Info("event system started successfully")

	// Create Kubernetes Operators/Controllers
	if cfg.EnableK8sControllers {
		// Initialise the controller runtime with our logger.
		ctrl.SetLogger(zapr.NewLogger(log.DefaultLogger.Desugar()))

		// Configure a scheme to include the required resource definitions.
		scheme := runtime.NewScheme()
		err = testworkflowsv1.AddToScheme(scheme)
		commons.ExitOnError("add TestWorkflows to kubernetes runtime scheme", err)

		// Legacy schemes
		err = testexecutionv1.AddToScheme(scheme)
		commons.ExitOnError("add TestExecution to kubernetes runtime scheme", err)
		err = testsuiteexecutionv1.AddToScheme(scheme)
		commons.ExitOnError("add TestSuiteExecution to kubernetes runtime scheme", err)

		// Configure the manager to use the defined scheme and to operate in the current namespace.
		mgr, err := manager.New(kubeConfig, manager.Options{
			Scheme: scheme,
			Cache: cache.Options{
				DefaultNamespaces: map[string]cache.Config{
					cfg.TestkubeNamespace: {},
				},
			},
		})
		commons.ExitOnError("creating kubernetes controller manager", err)

		// Initialise controllers
		err = controller.NewTestWorkflowExecutionExecutorController(mgr, testWorkflowExecutor)
		commons.ExitOnError("creating TestWorkflowExecution controller", err)

		// Legacy controllers
		testExecutor := workerpool.New[testkube.Test, testkube.ExecutionRequest, testkube.Execution](scheduler.DefaultConcurrencyLevel)
		err = controller.NewTestExecutionExecutorController(mgr, testExecutor, deprecatedSystem)
		commons.ExitOnError("creating TestExecution controller", err)
		testSuiteExecutor := workerpool.New[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution](scheduler.DefaultConcurrencyLevel)
		err = controller.NewTestSuiteExecutionExecutorController(mgr, testSuiteExecutor, deprecatedSystem)
		commons.ExitOnError("creating TestSuiteExecution controller", err)

		// Finally start the manager.
		g.Go(func() error {
			return mgr.Start(ctx)
		})
	}

	// Create HTTP server
	log.DefaultLogger.Infow("creating HTTP server...", "port", cfg.APIServerPort)
	httpServer := server.NewServer(server.Config{Port: cfg.APIServerPort, EnableTracing: cfg.TracingEnabled})
	httpServer.Routes.Use(cors.New())

	if deprecatedSystem != nil && deprecatedSystem.API != nil {
		deprecatedSystem.API.Init(httpServer)
	}

	isStandalone := mode == common.ModeStandalone
	var executionController scheduling.Controller
	if isStandalone && controlPlane != nil {
		executionController = controlPlane.ExecutionController
	}
	api := apiv1.NewTestkubeAPI(
		isStandalone,
		executionController,
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
		commons.ExitOnError("starting agent", err)
		g.Go(func() error {
			err = agentHandle.Run(ctx)
			commons.ExitOnError("running agent", err)
			return nil
		})
		eventsEmitter.Loader.Register(agentHandle)
	}

	if !cfg.DisableTestTriggers {
		k8sCfg, err := k8sclient.GetK8sClientConfig()
		commons.ExitOnError("getting k8s client config", err)
		testkubeClientset, err := testkubeclientset.NewForConfig(k8sCfg)
		commons.ExitOnError("creating TestKube Clientset", err)
		// TODO: Check why this simpler options is not working
		//testkubeClientset := testkubeclientset.New(clientset.RESTClient())

		var lb leasebackend.Repository
		if controlPlane != nil {
			lb = controlPlane.GetRepositoryManager().LeaseBackend()
		} else {
			// Fallback: Kubernetes Lease-based coordination (no external DB required)
			lb = leasebackendk8s.NewK8sLeaseBackend(clientset, cfg.TestkubeNamespace)
		}

		triggerService := triggers.NewService(
			cfg.RunnerName,
			deprecatedSystem,
			clientset,
			testkubeClientset,
			testWorkflowsClient,
			testTriggersClient,
			lb,
			log.DefaultLogger,
			configMapConfig,
			eventBus,
			metrics,
			executionWorker,
			testWorkflowExecutor,
			testWorkflowResultsRepository,
			&proContext,
			triggers.WithHostnameIdentifier(),
			triggers.WithTestkubeNamespace(cfg.TestkubeNamespace),
			triggers.WithWatcherNamespaces(cfg.TestkubeWatcherNamespaces),
			triggers.WithDisableSecretCreation(!secretConfig.AutoCreate),
			triggers.WithTestTriggerControlPlane(cfg.TestTriggerControlPlane),
			triggers.WithEventLabels(cfg.EventLabels),
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
		"testkube API Server started successfully",
		"telemetryEnabled", telemetryEnabled,
		"clusterId", clusterId,
		"namespace", cfg.TestkubeNamespace,
		"executionNamespace", cfg.DefaultExecutionNamespace,
		"version", version.Version,
		"startupTime", time.Since(startTime),
	)
	log.DefaultLogger.Infow("api endpoints ready", "httpPort", cfg.APIServerPort, "grpcPort", cfg.GRPCServerPort)

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
				log.DefaultLogger.Errorw("unable to parse label selector", "error", err, "label", "testkube="+resource)
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
				log.DefaultLogger.Errorw("unable to delete legacy cronjobs", "error", err, "label", "testkube="+resource, "namespace", cfg.TestkubeNamespace)
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
		log.DefaultLogger.Infow("http server starting...", "port", cfg.APIServerPort)
		return httpServer.Run(ctx)
	})

	if deprecatedSystem != nil {
		if deprecatedSystem.Reconciler != nil {
			g.Go(func() error {
				return deprecatedSystem.Reconciler.Run(ctx)
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
