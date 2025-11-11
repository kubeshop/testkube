package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-logr/zapr"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	k8sctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	executorv1 "github.com/kubeshop/testkube/api/executor/v1"
	testexecutionv1 "github.com/kubeshop/testkube/api/testexecution/v1"
	testsuiteexecutionv1 "github.com/kubeshop/testkube/api/testsuiteexecution/v1"
	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/cmd/api-server/services"
	"github.com/kubeshop/testkube/internal/app/api/debug"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/internal/common"
	syncagent "github.com/kubeshop/testkube/internal/sync"
	synccontroller "github.com/kubeshop/testkube/internal/sync/controller"
	syncgrpc "github.com/kubeshop/testkube/internal/sync/grpc"
	"github.com/kubeshop/testkube/pkg/agent"
	agentclient "github.com/kubeshop/testkube/pkg/agent/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudartifacts "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	cloudwebhook "github.com/kubeshop/testkube/pkg/cloud/data/webhook"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/controller"
	"github.com/kubeshop/testkube/pkg/controlplane"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/coordination/leader"
	"github.com/kubeshop/testkube/pkg/cronjob"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/event/kind/cdevent"
	"github.com/kubeshop/testkube/pkg/event/kind/k8sevent"
	"github.com/kubeshop/testkube/pkg/event/kind/testworkflowexecutionmetrics"
	"github.com/kubeshop/testkube/pkg/event/kind/testworkflowexecutions"
	"github.com/kubeshop/testkube/pkg/event/kind/testworkflowexecutiontelemetry"
	"github.com/kubeshop/testkube/pkg/event/kind/webhook"
	ws "github.com/kubeshop/testkube/pkg/event/kind/websocket"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testtriggerclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	observtracing "github.com/kubeshop/testkube/pkg/observability/tracing"
	kubeclient "github.com/kubeshop/testkube/pkg/operator/client"
	executorsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/executors/v1"
	testtriggersclientv1 "github.com/kubeshop/testkube/pkg/operator/client/testtriggers/v1"
	testworkflowsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/testworkflows/v1"
	testkubeclientset "github.com/kubeshop/testkube/pkg/operator/clientset/versioned"
	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
	leasebackendk8s "github.com/kubeshop/testkube/pkg/repository/leasebackend/k8s"
	runner2 "github.com/kubeshop/testkube/pkg/runner"
	runnergrpc "github.com/kubeshop/testkube/pkg/runner/grpc"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/kubeshop/testkube/pkg/tcl/schedulertcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/presets"
	"github.com/kubeshop/testkube/pkg/triggers"
	"github.com/kubeshop/testkube/pkg/version"
	"github.com/kubeshop/testkube/pkg/workerpool"
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
	// Determine the running mode

	mode := common.ModeAgent
	if cfg.TestkubeProAPIKey == "" && cfg.TestkubeProAgentRegToken == "" {
		mode = common.ModeStandalone
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

	log.DefaultLogger.Info("connecting to NATS...")
	nc := commons.MustCreateNATSConnection(cfg)
	log.DefaultLogger.Infow("connected to NATS successfully", "embedded", cfg.NatsEmbedded, "uri", cfg.NatsURI)

	eventBus := bus.NewNATSBus(nc)
	if cfg.Trace {
		eventBus.TraceEvents()
	}
	// TODO(emil): do we need a mongo/postgres backend for leases like for test triggers?
	eventsEmitterLeaseBackend := leasebackendk8s.NewK8sLeaseBackend(clientset, "testkube-agent-"+cfg.APIServerFullname, cfg.TestkubeNamespace)
	eventsEmitter := event.NewEmitter(eventBus, eventsEmitterLeaseBackend, "agentevents", cfg.TestkubeClusterName)

	// Connect to the Control Plane
	var grpcConn *grpc.ClientConn
	var controlPlane *controlplane.Server
	if mode == common.ModeStandalone {
		log.DefaultLogger.Info("starting embedded Control Plane service...")
		controlPlane = services.CreateControlPlane(ctx, cfg, features, eventsEmitter)

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

	var leaderLeaseBackend leasebackend.Repository
	if controlPlane != nil {
		leaderLeaseBackend = controlPlane.GetRepositoryManager().LeaseBackend()
	} else {
		leaderLeaseBackend = leasebackendk8s.NewK8sLeaseBackend(
			clientset,
			"testkube-core",
			cfg.TestkubeNamespace,
			leasebackendk8s.WithLeaseName(cfg.TestkubeLeaseName),
		)
	}

	leaderTasks := make([]leader.Task, 0)

	// Agents should be able to make an initial registration with their
	// registration token and subsequent registrations with their API key to
	// enable them to update their set of capabilities.
	// To transition super agents to the new capabilities based registration,
	// the charts have been changed to set the old super agent API key value as
	// their registration token and registration has been enabled with the use
	// of the API key. After intial self registration the super agent will be
	// able to use the API key in the generated secret to reregister.
	if cfg.TestkubeProAPIKey != "" || cfg.TestkubeProAgentRegToken != "" {
		runnerName := cfg.RunnerName
		if runnerName == "" {
			// Fallback to a set name, but this is unlikely to be unique.
			// Note, that the runner name for a super agent will be overriden
			// on the backend to make sure it is unique within the org.
			runnerName = cfg.APIServerFullname
		}
		log.DefaultLogger.Infow("registering runner", "runner_name", runnerName)

		// Build capabilities based on enabled features
		capabilities := []cloud.AgentCapability{}
		if !cfg.DisableRunner {
			capabilities = append(capabilities, cloud.AgentCapability_AGENT_CAPABILITY_RUNNER)
		}
		if !cfg.DisableTestTriggers {
			capabilities = append(capabilities, cloud.AgentCapability_AGENT_CAPABILITY_LISTENER)
		}
		if !cfg.DisableWebhooks {
			if cfg.EnableCloudWebhooks {
				// The presence of an agent with this capability within an
				// environment toggles Webhooks for the environment from
				// being emitted by the agent to being emitted by the
				// control plane.
				capabilities = append(capabilities, cloud.AgentCapability_AGENT_CAPABILITY_CLOUD_WEBHOOKS)
			} else {
				capabilities = append(capabilities, cloud.AgentCapability_AGENT_CAPABILITY_WEBHOOKS)
			}
		}

		// Get all labels that matches with prefix
		runnerLabels := getDeploymentLabels(ctx, clientset, cfg.TestkubeNamespace, cfg.APIServerFullname, cfg.RunnerLabelsPrefix)
		runnerLabels["registration"] = "self"

		// Debug labels found
		log.DefaultLogger.Debugw("labels to be configured", runnerLabels)

		registrationToken := cfg.TestkubeProAgentRegToken
		if cfg.TestkubeProAPIKey != "" {
			registrationToken = cfg.TestkubeProAPIKey
		}

		res, err := grpcClient.Register(ctx, &cloud.RegisterRequest{
			RegistrationToken: registrationToken,
			RunnerName:        runnerName,
			OrganizationId:    cfg.TestkubeProOrgID,
			Floating:          cfg.FloatingRunner,
			Capabilities:      capabilities,
			RunnerGroup:       cfg.RunnerGroup,
			IsGlobal:          cfg.IsGlobal,
			Labels:            runnerLabels,
		})
		if err != nil {
			log.DefaultLogger.Fatalw("error registering runner", "error", err.Error())
		}
		log.DefaultLogger.Infow("registered runner", "runner_name", runnerName, "runner_id", res.RunnerId, "organization_id", res.OrganizationId)

		// Add the new values to the current configuration.
		cfg.TestkubeProAPIKey = res.RunnerKey
		cfg.TestkubeProAgentID = res.RunnerId
		cfg.TestkubeProOrgID = res.OrganizationId

		// Attempt to store the values in a Kubernetes secret for consumption next startup.
		if cfg.SelfRegistrationSecret == "" {
			log.DefaultLogger.Warnw("unable to save api key from registration with the self registration secret unspecified, will reuse registration token for subsequent deployments")
		} else if cfg.DisableSecretCreation {
			log.DefaultLogger.Warnw("unable to save api key from registration with secrets disabled, will reuse registration token for subsequent deployments")
		} else {
			// Create or update the existing secret
			_, err := secretManager.Get(ctx, cfg.TestkubeNamespace, cfg.SelfRegistrationSecret)
			secretData := map[string]string{
				"TESTKUBE_PRO_API_KEY":  res.RunnerKey,
				"TESTKUBE_PRO_AGENT_ID": res.RunnerId,
				"TESTKUBE_PRO_ORG_ID":   res.OrganizationId,
			}
			if errors.Is(err, secretmanager.ErrNotFound) {
				if _, err := secretManager.Create(ctx, cfg.TestkubeNamespace, cfg.SelfRegistrationSecret, secretData, secretmanager.CreateOptions{Bypass: true}); err != nil {
					log.DefaultLogger.Errorw("error creating self-register runner secret", "error", err.Error())
				} else {
					log.DefaultLogger.Infow("saved registration in secret", "runner_name", runnerName, "secret_name", cfg.SelfRegistrationSecret)
				}
			} else {
				if _, err := secretManager.Update(ctx, cfg.TestkubeNamespace, cfg.SelfRegistrationSecret, secretData, secretmanager.UpdateOptions{Bypass: true}); err != nil {
					log.DefaultLogger.Errorw("error updating self-register runner secret", "error", err.Error())
				} else {
					log.DefaultLogger.Infow("updated registration in secret", "runner_name", runnerName, "secret_name", cfg.SelfRegistrationSecret)
				}
			}
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
			// Here we are not using capabilities because the client and/or server may not have TLS implemented as it was not previously
			// enforced, however it is required with the new client implementation to secure authentication tokens.
			// gRPC does not provide a specific error for an attempt to transmit credentials over an unencrypted connection so to
			// prevent the fallback to the previous insecure behaviour we must instead check to see whether connectivity can be
			// established. The repercussions of this is that the agent will eagerly fallback to the insecure and legacy behaviour
			// and so bringing up an agent before networking with the Control Plane has been established, or during a Control Plane
			// outage will cause a fallback to the previous behaviour.
			// This check should be removed once TLS is enforced across deployments.
			if !runnerClient.IsSupported(ctx, proContext.EnvID) {
				log.DefaultLogger.Warn("new runner RPC is not supported by Control Plane, falling back to previous implementation.")
				return runnerService.Start(ctx, true)
			}
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
	if !cfg.DisableWebhooks && !cfg.EnableCloudWebhooks {
		secretClient := secret.NewClientFor(clientset, cfg.TestkubeNamespace)
		webhookLoader := webhook.NewWebhookLoader(
			webhooksClient,
			webhook.WithDeprecatedClients(deprecatedClients),
			webhook.WithDeprecatedRepositories(deprecatedRepositories),
			webhook.WithTestWorkflowResultsRepository(testWorkflowResultsRepository),
			webhook.WithWebhookResultsRepository(webhookRepository),
			webhook.WithWebhookTemplateClient(webhookTemplatesClient),
			webhook.WithSecretClient(secretClient),
			webhook.WithMetrics(metrics),
			webhook.WithEnvs(envs),
			webhook.WithProContext(&proContext))
		eventsEmitter.RegisterLoader(webhookLoader)
	}
	websocketLoader := ws.NewWebsocketLoader()
	eventsEmitter.RegisterLoader(websocketLoader)
	eventsEmitter.RegisterLoader(commons.MustCreateSlackLoader(cfg, envs))
	if cfg.CDEventsTarget != "" {
		cdeventLoader, err := cdevent.NewCDEventLoader(cfg.CDEventsTarget, clusterId, cfg.TestkubeNamespace, proContext.DashboardURI, testkube.AllEventTypes)
		if err == nil {
			eventsEmitter.RegisterLoader(cdeventLoader)
		} else {
			log.DefaultLogger.Debugw("cdevents init error", "error", err.Error())
		}
	}
	if cfg.EnableK8sEvents {
		eventsEmitter.RegisterLoader(k8sevent.NewK8sEventLoader(clientset, cfg.TestkubeNamespace, testkube.AllEventTypes))
	}

	// Update the Prometheus metrics regarding the Test Workflow Execution
	eventsEmitter.RegisterLoader(testworkflowexecutionmetrics.NewLoader(ctx, metrics, proContext.DashboardURI))

	// Send the telemetry data regarding the Test Workflow Execution
	// TODO: Disable it if Control Plane does that
	eventsEmitter.RegisterLoader(testworkflowexecutiontelemetry.NewLoader(ctx, configMapConfig))

	// Update TestWorkflowExecution Kubernetes resource objects on status change
	eventsEmitter.RegisterLoader(testworkflowexecutions.NewLoader(ctx, cfg.TestkubeNamespace, kubeClient))

	g.Go(func() error {
		eventsEmitter.Listen(ctx)
		return nil
	})

	/////////////////////////////////
	// KUBERNETES CONTROLLER SETUP
	if cfg.EnableK8sControllers || cfg.GitOpsSyncKubernetesToCloudEnabled {
		// Initialise the controller runtime with our logger.
		ctrl.SetLogger(zapr.NewLogger(log.DefaultLogger.Desugar()))

		// Configure a scheme to include the required resource definitions.
		scheme := runtime.NewScheme()
		err = testworkflowsv1.AddToScheme(scheme)
		commons.ExitOnError("add TestWorkflows to kubernetes runtime scheme", err)
		err = testtriggersv1.AddToScheme(scheme)
		commons.ExitOnError("add TestTriggers to kubernetes runtime scheme", err)
		err = executorv1.AddToScheme(scheme)
		commons.ExitOnError("add Webhooks to kubernetes runtime scheme", err)

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

		// Create Sync Controllers
		if proContext.CloudStorageSupportedInControlPlane && cfg.GitOpsSyncKubernetesToCloudEnabled {
			var store interface {
				synccontroller.TestTriggerStore
				synccontroller.TestWorkflowStore
				synccontroller.TestWorkflowTemplateStore
				synccontroller.WebhookStore
				synccontroller.WebhookTemplateStore
			}
			store = syncgrpc.NewClient(grpcConn, log.DefaultLogger, proContext.APIKey, proContext.OrgID)
			// If the agent is running without secure gRPC TLS connection to the Control Plane then the client will not be able to
			// connect and so we need to fallback to an implementation that doesn't do anything.
			if cfg.TestkubeProTLSInsecure || cfg.TestkubeProSkipVerify {
				log.DefaultLogger.Warn("Unable to create GitOps sync connection to Control Plane when running in insecure TLS mode. Kubernetes resource updates will not be synced with the Control Plane!")
				store = syncagent.NoOpStore{}
			}

			err = synccontroller.NewTestTriggerSyncController(mgr, store)
			commons.ExitOnError("creating TestTrigger sync controller", err)
			err = synccontroller.NewTestWorkflowSyncController(mgr, store)
			commons.ExitOnError("creating TestWorkflow sync controller", err)
			err = synccontroller.NewTestWorkflowTemplateSyncController(mgr, store)
			commons.ExitOnError("creating TestWorkflowTemplate sync controller", err)
			err = synccontroller.NewWebhookSyncController(mgr, store)
			commons.ExitOnError("creating Webhook sync controller", err)
			err = synccontroller.NewWebhookTemplateSyncController(mgr, store)
			commons.ExitOnError("creating WebhookTemplate sync controller", err)
		}

		// Initialise controllers
		if cfg.EnableK8sControllers {
			err = controller.NewTestWorkflowExecutionExecutorController(mgr, testWorkflowExecutor)
			commons.ExitOnError("creating TestWorkflowExecution controller", err)

			// Legacy controllers
			testExecutor := workerpool.New[testkube.Test, testkube.ExecutionRequest, testkube.Execution](scheduler.DefaultConcurrencyLevel)
			err = controller.NewTestExecutionExecutorController(mgr, testExecutor, deprecatedSystem)
			commons.ExitOnError("creating TestExecution controller", err)
			testSuiteExecutor := workerpool.New[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution](scheduler.DefaultConcurrencyLevel)
			err = controller.NewTestSuiteExecutionExecutorController(mgr, testSuiteExecutor, deprecatedSystem)
			commons.ExitOnError("creating TestSuiteExecution controller", err)
		}

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
		leaderTasks = append(leaderTasks, leader.Task{
			Name: "agent",
			Start: func(taskCtx context.Context) error {
				err := agentHandle.Run(taskCtx)
				if err != nil && !errors.Is(err, context.Canceled) {
					commons.ExitOnError("running agent", err)
				}
				return err
			},
		})
		eventsEmitter.Register(agentHandle)
	}

	if !cfg.DisableTestTriggers {
		k8sCfg, err := k8sclient.GetK8sClientConfig()
		commons.ExitOnError("getting k8s client config", err)
		testkubeClientset, err := testkubeclientset.NewForConfig(k8sCfg)
		commons.ExitOnError("creating TestKube Clientset", err)
		// TODO: Check why this simpler options is not working
		// testkubeClientset := testkubeclientset.New(clientset.RESTClient())

		var triggersLeaseBackend leasebackend.Repository
		if controlPlane != nil {
			triggersLeaseBackend = controlPlane.GetRepositoryManager().LeaseBackend()
		} else {
			// Fallback: Kubernetes Lease-based coordination (no external DB required)
			triggersLeaseBackend = leasebackendk8s.NewK8sLeaseBackend(
				clientset,
				"testkube-triggers-lease",
				cfg.TestkubeNamespace,
				leasebackendk8s.WithLeaseName(cfg.TestkubeLeaseName),
			)
		}

		triggerService := triggers.NewService(
			cfg.RunnerName,
			deprecatedSystem,
			clientset,
			testkubeClientset,
			testWorkflowsClient,
			testTriggersClient,
			triggersLeaseBackend,
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
	leaderTasks = append(leaderTasks, leader.Task{
		Name: "telemetry-heartbeat",
		Start: func(taskCtx context.Context) error {
			services.HandleTelemetryHeartbeat(taskCtx, clusterId, configMapConfig)
			return nil
		},
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
	log.DefaultLogger.Infow("api endpoints ready",
		"httpPort", cfg.APIServerPort,
		"grpcPort", cfg.GRPCServerPort,
	)

	if cfg.EnableDebugServer {
		debugSrv := debug.NewDebugServer(cfg.DebugListenAddr)

		g.Go(func() error {
			log.DefaultLogger.Infof("starting debug pprof server")
			return debugSrv.ListenAndServe()
		})
	}

	scheduler := commons.CreateCronJobScheduler(
		cfg,
		testWorkflowsClient,
		testWorkflowTemplatesClient,
		testWorkflowExecutor,
		log.DefaultLogger,
		&proContext,
	)
	if scheduler != nil {
		// Remove any remaining legacy cronjobs.
		// TODO: Remove this section once we are happy that users are not migrating from legacy cronjobs (next Major version?)
		resources := []string{cronjob.TestResourceURI, cronjob.TestSuiteResourceURI, cronjob.TestWorkflowResourceURI}
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
		leaderTasks = append(leaderTasks, leader.Task{
			Name: "cron-scheduler",
			Start: func(taskCtx context.Context) error {
				scheduler.Reconcile(taskCtx)
				return nil
			},
		})
	}

	g.Go(func() error {
		log.DefaultLogger.Infow("http server starting...", "port", cfg.APIServerPort)
		return httpServer.Run(ctx)
	})

	if len(leaderTasks) > 0 {
		leaderIdentifier := resolveLeaderIdentifier()

		leaderClusterID := clusterId
		if leaderClusterID == "" {
			leaderClusterID = "testkube-core"
		} else {
			leaderClusterID = fmt.Sprintf("%s-core", leaderClusterID)
		}

		coordinatorLogger := log.DefaultLogger.With("component", "leader-coordinator")
		leaderCoordinator := leader.New(leaderLeaseBackend, leaderIdentifier, leaderClusterID, coordinatorLogger)
		for _, task := range leaderTasks {
			leaderCoordinator.Register(task)
		}

		g.Go(func() error {
			return leaderCoordinator.Run(ctx)
		})
	}

	if err := g.Wait(); err != nil {
		log.DefaultLogger.Fatalf("Testkube is shutting down: %v", err)
	}
}

func resolveLeaderIdentifier() string {
	if podName := os.Getenv("POD_NAME"); podName != "" {
		return podName
	}

	if host, err := os.Hostname(); err == nil && host != "" {
		return host
	}

	return fmt.Sprintf("testkube-core-%d", time.Now().UnixNano())
}

func getDeploymentLabels(ctx context.Context, clientset kubernetes.Interface, namespace, deploymentName string, labelPrefix string) map[string]string {
	if deploymentName == "" {
		return nil
	}

	deploy, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		log.DefaultLogger.Warnw("cannot read deployment labels", "deployment", deploymentName, "error", err.Error())
		return nil
	}

	// clone to avoid sharing internal maps
	labels := make(map[string]string, len(deploy.Labels))
	for k, v := range deploy.Labels {
		if strings.HasPrefix(k, labelPrefix) {
			shortKey := strings.TrimPrefix(k, labelPrefix)
			if shortKey != "" {
				labels[shortKey] = v
			}
		}
	}
	return labels
}