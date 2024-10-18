package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/cmd/api-server/services"
	"github.com/kubeshop/testkube/internal/app/api/debug"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/event/kind/cdevent"
	"github.com/kubeshop/testkube/pkg/event/kind/k8sevent"
	"github.com/kubeshop/testkube/pkg/event/kind/webhook"
	ws "github.com/kubeshop/testkube/pkg/event/kind/websocket"
	testworkflow2 "github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/tcl/checktcl"
	"github.com/kubeshop/testkube/pkg/tcl/schedulertcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/presets"

	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"

	cloudartifacts "github.com/kubeshop/testkube/pkg/cloud/data/artifact"

	domainstorage "github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/slack"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	dbmigrations "github.com/kubeshop/testkube/internal/db-migrations"
	parser "github.com/kubeshop/testkube/internal/template"
	"github.com/kubeshop/testkube/pkg/version"

	"golang.org/x/sync/errgroup"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/sequence"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	kubeexecutor "github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/containerexecutor"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/scheduler"

	testkubeclientset "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/triggers"

	kubeclient "github.com/kubeshop/testkube-operator/pkg/client"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/dbmigrator"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/reconciler"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

func init() {
	flag.Parse()
}

func runMongoMigrations(ctx context.Context, db *mongo.Database, _ string) error {
	migrationsCollectionName := "__migrations"
	activeMigrations, err := dbmigrator.GetDbMigrationsFromFs(dbmigrations.MongoMigrationsFs)
	if err != nil {
		return errors.Wrap(err, "failed to obtain MongoDB migrations from disk")
	}
	dbMigrator := dbmigrator.NewDbMigrator(dbmigrator.NewDatabase(db, migrationsCollectionName), activeMigrations)
	plan, err := dbMigrator.Plan(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to plan MongoDB migrations")
	}
	if plan.Total == 0 {
		log.DefaultLogger.Info("No MongoDB migrations to apply.")
	} else {
		log.DefaultLogger.Info(fmt.Sprintf("Applying MongoDB migrations: %d rollbacks and %d ups.", len(plan.Downs), len(plan.Ups)))
	}
	err = dbMigrator.Apply(ctx)
	return errors.Wrap(err, "failed to apply MongoDB migrations")
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
	commons.HandleCancelSignal(g, ctx)

	commons.MustFreePort(cfg.APIServerPort)
	commons.MustFreePort(cfg.GraphqlPort)

	kubeClient, err := kubeclient.GetClient()
	exitOnError("Getting kubernetes client", err)

	secretClient, err := secret.NewClient(cfg.TestkubeNamespace)
	exitOnError("Getting secret client", err)

	configMapClient, err := configmap.NewClient(cfg.TestkubeNamespace)
	exitOnError("Getting config map client", err)

	// agent
	var grpcClient cloud.TestKubeCloudAPIClient
	var grpcConn *grpc.ClientConn
	if mode == common.ModeAgent {
		grpcConn, err = agent.NewGRPCConnection(
			ctx,
			cfg.TestkubeProTLSInsecure,
			cfg.TestkubeProSkipVerify,
			cfg.TestkubeProURL,
			cfg.TestkubeProCertFile,
			cfg.TestkubeProKeyFile,
			cfg.TestkubeProCAFile, //nolint
			log.DefaultLogger,
		)
		exitOnError("error creating gRPC connection", err)
		defer grpcConn.Close()

		grpcClient = cloud.NewTestKubeCloudAPIClient(grpcConn)
	}

	// k8s
	deprecatedClients := commons.CreateDeprecatedClients(kubeClient, cfg.TestkubeNamespace)
	webhooksClient := executorsclientv1.NewWebhooksClient(kubeClient, cfg.TestkubeNamespace)
	testWorkflowExecutionsClient := testworkflowsclientv1.NewTestWorkflowExecutionsClient(kubeClient, cfg.TestkubeNamespace)

	var testWorkflowsClient testworkflowsclientv1.Interface
	var testWorkflowTemplatesClient testworkflowsclientv1.TestWorkflowTemplatesInterface
	if mode == common.ModeAgent && cfg.WorkflowStorage == "control-plane" {
		testWorkflowsClient = cloudtestworkflow.NewCloudTestWorkflowRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
		testWorkflowTemplatesClient = cloudtestworkflow.NewCloudTestWorkflowTemplateRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
	} else {
		testWorkflowsClient = testworkflowsclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)
		testWorkflowTemplatesClient = testworkflowsclientv1.NewTestWorkflowTemplatesClient(kubeClient, cfg.TestkubeNamespace)
	}

	clientset, err := k8sclient.ConnectToK8s()
	exitOnError("Creating k8s clientset", err)
	k8sCfg, err := k8sclient.GetK8sClientConfig()
	exitOnError("Getting k8s client config", err)
	testkubeClientset, err := testkubeclientset.NewForConfig(k8sCfg)
	exitOnError("Creating TestKube Clientset", err)

	var logGrpcClient logsclient.StreamGetter
	if features.LogsV2 {
		logGrpcClient = commons.MustGetLogsV2Client(cfg)
	}

	// DI
	var deprecatedRepositories commons.DeprecatedRepositories
	var testWorkflowResultsRepository testworkflow2.Repository
	var testWorkflowOutputRepository testworkflow2.OutputRepository
	var triggerLeaseBackend triggers.LeaseBackend
	var artifactStorage domainstorage.ArtifactsStorage
	var storageClient domainstorage.Client
	if mode == common.ModeAgent {
		deprecatedRepositories = commons.CreateDeprecatedRepositoriesForCloud(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
		testWorkflowResultsRepository = cloudtestworkflow.NewCloudRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
		var opts []cloudtestworkflow.Option
		if cfg.StorageSkipVerify {
			opts = append(opts, cloudtestworkflow.WithSkipVerify())
		}
		testWorkflowOutputRepository = cloudtestworkflow.NewCloudOutputRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey, opts...)
		triggerLeaseBackend = triggers.NewAcquireAlwaysLeaseBackend()
		artifactStorage = cloudartifacts.NewCloudArtifactsStorage(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
	} else {
		// Connect to storages
		db := commons.MustGetMongoDatabase(cfg, secretClient)
		storageClient = commons.MustGetMinioClient(cfg)

		// Build repositories
		sequenceRepository := sequence.NewMongoRepository(db)
		testWorkflowResultsRepository = testworkflow2.NewMongoRepository(db, cfg.APIMongoAllowDiskUse,
			testworkflow2.WithMongoRepositorySequence(sequenceRepository))
		triggerLeaseBackend = triggers.NewMongoLeaseBackend(db)
		testWorkflowOutputRepository = testworkflow2.NewMinioOutputRepository(storageClient, db.Collection(testworkflow2.CollectionName), cfg.LogsBucket)
		artifactStorage = minio.NewMinIOArtifactClient(storageClient)
		deprecatedRepositories = commons.CreateDeprecatedRepositoriesForMongo(ctx, cfg, db, logGrpcClient, storageClient, features)

		// Run DB migrations
		if !cfg.DisableMongoMigrations {
			err := runMongoMigrations(ctx, db, filepath.Join(cfg.TestkubeConfigDir, "db-migrations"))
			if err != nil {
				log.DefaultLogger.Warnf("failed to apply MongoDB migrations: %v", err)
			}
		}
	}

	configMapConfig := commons.MustGetConfigMapConfig(ctx, cfg.APIServerConfig, cfg.TestkubeNamespace, cfg.TestkubeAnalyticsEnabled)
	clusterId, _ := configMapConfig.GetUniqueClusterId(context.Background())
	telemetryEnabled, _ := configMapConfig.GetTelemetryEnabled(context.Background())

	apiVersion := version.Version

	// TODO: Check why these are needed at all
	envs := commons.GetEnvironmentVariables()

	nc, err := newNATSEncodedConnection(cfg)
	exitOnError("Creating NATS connection", err)

	eventBus := bus.NewNATSBus(nc)
	if cfg.Trace {
		eventBus.TraceEvents()
	}

	eventsEmitter := event.NewEmitter(eventBus, cfg.TestkubeClusterName)

	var logsStream logsclient.Stream

	if features.LogsV2 {
		logsStream, err = logsclient.NewNatsLogStream(nc.Conn)
		exitOnError("Creating logs streaming client", err)
	}

	metrics := metrics.NewMetrics()

	defaultExecutors, images, err := parseDefaultExecutors(cfg)
	exitOnError("Parsing default executors", err)
	if !cfg.TestkubeReadonlyExecutors {
		err := kubeexecutor.SyncDefaultExecutors(deprecatedClients.Executors(), cfg.TestkubeNamespace, defaultExecutors)
		exitOnError("Sync default executors", err)
	}

	proContext := newProContext(cfg, grpcClient)

	// Check Pro/Enterprise subscription
	var subscriptionChecker checktcl.SubscriptionChecker
	if mode == common.ModeAgent {
		subscriptionChecker, err = checktcl.NewSubscriptionChecker(ctx, proContext, grpcClient, grpcConn)
		exitOnError("Failed creating subscription checker", err)
	}

	serviceAccountNames := map[string]string{
		cfg.TestkubeNamespace: cfg.JobServiceAccountName,
	}

	// Pro edition only (tcl protected code)
	if cfg.TestkubeExecutionNamespaces != "" {
		err = subscriptionChecker.IsActiveOrgPlanEnterpriseForFeature("execution namespace")
		exitOnError("Subscription checking", err)

		serviceAccountNames = schedulertcl.GetServiceAccountNamesFromConfig(serviceAccountNames, cfg.TestkubeExecutionNamespaces)
	}

	jobTemplates, err := parser.ParseJobTemplates(cfg)
	exitOnError("Creating job templates", err)
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
		"http://"+cfg.APIServerFullname+":"+cfg.APIServerPort,
		cfg.NatsURI,
		cfg.Debug,
		logsStream,
		features,
		cfg.TestkubeDefaultStorageClassName,
		cfg.WhitelistedContainers,
	)
	exitOnError("Creating executor client", err)

	inspector := commons.CreateImageInspector(cfg, configMapClient, secretClient)

	containerTemplates, err := parser.ParseContainerTemplates(cfg)
	exitOnError("Creating container job templates", err)
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
		"http://"+cfg.APIServerFullname+":"+cfg.APIServerPort,
		cfg.NatsURI,
		cfg.Debug,
		logsStream,
		features,
		cfg.TestkubeDefaultStorageClassName,
		cfg.WhitelistedContainers,
		cfg.TestkubeImageCredentialsCacheTTL,
	)
	exitOnError("Creating container executor", err)

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
	)
	if mode == common.ModeAgent {
		sched.WithSubscriptionChecker(subscriptionChecker)
	}

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

	testWorkflowProcessor := presets.NewOpenSource(inspector)
	if mode == common.ModeAgent {
		testWorkflowProcessor = presets.NewPro(inspector)
	}

	// Build internal execution worker
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

	go testWorkflowExecutor.Recover(context.Background())

	slackLoader, err := newSlackLoader(cfg, envs)
	exitOnError("Creating slack loader", err)

	// Initialize event handlers
	websocketLoader := ws.NewWebsocketLoader()
	eventsEmitter.Loader.Register(webhook.NewWebhookLoader(log.DefaultLogger, webhooksClient, deprecatedClients.Templates(), deprecatedRepositories.TestResults(), deprecatedRepositories.TestSuiteResults(), testWorkflowResultsRepository, metrics, &proContext, envs))
	eventsEmitter.Loader.Register(websocketLoader)
	eventsEmitter.Loader.Register(slackLoader)
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
	g.Go(func() error {
		eventsEmitter.Reconcile(ctx)
		return nil
	})
	eventsEmitter.Listen(ctx)

	api := apiv1.NewTestkubeAPI(
		deprecatedRepositories,
		deprecatedClients,
		cfg.TestkubeNamespace,
		testWorkflowResultsRepository,
		testWorkflowOutputRepository,
		secretClient,
		secretManager,
		webhooksClient,
		clientset,
		testkubeClientset,
		testWorkflowsClient,
		testWorkflowTemplatesClient,
		configMapConfig,
		clusterId,
		eventsEmitter,
		websocketLoader,
		executor,
		containerExecutor,
		testWorkflowExecutor,
		executionWorker,
		metrics,
		sched,
		slackLoader,
		cfg.GraphqlPort,
		artifactStorage,
		cfg.TestkubeDashboardURI,
		cfg.TestkubeHelmchartVersion,
		mode,
		eventBus,
		secretConfig,
		features,
		logsStream,
		logGrpcClient,
		serviceAccountNames,
		envs,
		cfg.TestkubeDockerImageVersion,
		&proContext,
	)

	if mode == common.ModeAgent {
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
			api.Mux.Handler(),
			grpcClient,
			api.GetLogsStream,
			getTestWorkflowNotificationsStream,
			clusterId,
			cfg.TestkubeClusterName,
			features,
			&proContext,
			cfg.TestkubeDockerImageVersion,
		)
		exitOnError("Starting agent", err)
		g.Go(func() error {
			err = agentHandle.Run(ctx)
			exitOnError("Running agent", err)
			return nil
		})
		eventsEmitter.Loader.Register(agentHandle)
	}

	api.Init()
	if !cfg.DisableTestTriggers {
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
		triggerService.Run(ctx)
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
	telemetryCh := make(chan struct{})
	defer close(telemetryCh)

	api.SendTelemetryStartEvent(ctx, telemetryCh)
	api.StartTelemetryHeartbeats(ctx, telemetryCh)

	log.DefaultLogger.Infow(
		"starting Testkube API server",
		"telemetryEnabled", telemetryEnabled,
		"clusterId", clusterId,
		"namespace", cfg.TestkubeNamespace,
		"version", apiVersion,
	)

	if cfg.EnableDebugServer {
		debugSrv := debug.NewDebugServer(cfg.DebugListenAddr)

		g.Go(func() error {
			log.DefaultLogger.Infof("starting debug pprof server")
			return debugSrv.ListenAndServe()
		})
	}

	g.Go(func() error {
		return api.Run(ctx)
	})

	g.Go(func() error {
		return api.RunGraphQLServer(ctx, cfg.GraphqlPort)
	})

	if err := g.Wait(); err != nil {
		log.DefaultLogger.Fatalf("Testkube is shutting down: %v", err)
	}
}

func parseDefaultExecutors(cfg *config.Config) (executors []testkube.ExecutorDetails, images kubeexecutor.Images, err error) {
	rawExecutors, err := parser.LoadConfigFromStringOrFile(
		cfg.TestkubeDefaultExecutors,
		cfg.TestkubeConfigDir,
		"executors.json",
		"executors",
	)
	if err != nil {
		return nil, images, err
	}

	if err = json.Unmarshal([]byte(rawExecutors), &executors); err != nil {
		return nil, images, err
	}

	enabledExecutors, err := parser.LoadConfigFromStringOrFile(
		cfg.TestkubeEnabledExecutors,
		cfg.TestkubeConfigDir,
		"enabledExecutors",
		"enabled executors",
	)
	if err != nil {
		return nil, images, err
	}

	// Load internal images
	next := make([]testkube.ExecutorDetails, 0)
	for i := range executors {
		if executors[i].Name == "logs-sidecar" {
			images.LogSidecar = executors[i].Executor.Image
			continue
		}
		if executors[i].Name == "init-executor" {
			images.Init = executors[i].Executor.Image
			continue
		}
		if executors[i].Name == "scraper-executor" {
			images.Scraper = executors[i].Executor.Image
			continue
		}
		if executors[i].Executor == nil {
			continue
		}
		next = append(next, executors[i])
	}
	executors = next

	// When there is no executors selected, enable all
	if enabledExecutors == "" {
		return executors, images, nil
	}

	// Filter enabled executors
	specifiedExecutors := make(map[string]struct{})
	for _, executor := range strings.Split(enabledExecutors, ",") {
		if strings.TrimSpace(executor) == "" {
			continue
		}
		specifiedExecutors[strings.TrimSpace(executor)] = struct{}{}
	}

	next = make([]testkube.ExecutorDetails, 0)
	for i := range executors {
		if _, ok := specifiedExecutors[executors[i].Name]; ok {
			next = append(next, executors[i])
		}
	}

	return next, images, nil
}

func newNATSEncodedConnection(cfg *config.Config) (*nats.EncodedConn, error) {
	// if embedded NATS server is enabled, we'll replace connection with one to the embedded server
	if cfg.NatsEmbedded {
		_, nc, err := event.ServerWithConnection(cfg.NatsEmbeddedStoreDir)
		if err != nil {
			return nil, err
		}

		log.DefaultLogger.Info("Started embedded NATS server")

		return nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	}

	return bus.NewNATSEncodedConnection(bus.ConnectionConfig{
		NatsURI:            cfg.NatsURI,
		NatsSecure:         cfg.NatsSecure,
		NatsSkipVerify:     cfg.NatsSkipVerify,
		NatsCertFile:       cfg.NatsCertFile,
		NatsKeyFile:        cfg.NatsKeyFile,
		NatsCAFile:         cfg.NatsCAFile,
		NatsConnectTimeout: cfg.NatsConnectTimeout,
	})
}

func newSlackLoader(cfg *config.Config, envs map[string]string) (*slack.SlackLoader, error) {
	slackTemplate, err := parser.LoadConfigFromStringOrFile(
		cfg.SlackTemplate,
		cfg.TestkubeConfigDir,
		"slack-template.json",
		"slack template",
	)
	if err != nil {
		return nil, err
	}

	slackConfig, err := parser.LoadConfigFromStringOrFile(cfg.SlackConfig, cfg.TestkubeConfigDir, "slack-config.json", "slack config")
	if err != nil {
		return nil, err
	}

	return slack.NewSlackLoader(slackTemplate, slackConfig, cfg.TestkubeClusterName, cfg.TestkubeDashboardURI,
		testkube.AllEventTypes, envs), nil
}

func newProContext(cfg *config.Config, grpcClient cloud.TestKubeCloudAPIClient) config.ProContext {
	proContext := config.ProContext{
		APIKey:                           cfg.TestkubeProAPIKey,
		URL:                              cfg.TestkubeProURL,
		TLSInsecure:                      cfg.TestkubeProTLSInsecure,
		WorkerCount:                      cfg.TestkubeProWorkerCount,
		LogStreamWorkerCount:             cfg.TestkubeProLogStreamWorkerCount,
		WorkflowNotificationsWorkerCount: cfg.TestkubeProWorkflowNotificationsWorkerCount,
		SkipVerify:                       cfg.TestkubeProSkipVerify,
		EnvID:                            cfg.TestkubeProEnvID,
		OrgID:                            cfg.TestkubeProOrgID,
		Migrate:                          cfg.TestkubeProMigrate,
		ConnectionTimeout:                cfg.TestkubeProConnectionTimeout,
		DashboardURI:                     cfg.TestkubeDashboardURI,
	}

	if grpcClient == nil {
		return proContext
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	md := metadata.Pairs("api-key", cfg.TestkubeProAPIKey)
	ctx = metadata.NewOutgoingContext(ctx, md)
	defer cancel()
	getProContext, err := grpcClient.GetProContext(ctx, &emptypb.Empty{})
	if err != nil {
		log.DefaultLogger.Warnf("cannot fetch pro-context from cloud: %s", err)
		return proContext
	}

	if proContext.EnvID == "" {
		proContext.EnvID = getProContext.EnvId
	}

	if proContext.OrgID == "" {
		proContext.OrgID = getProContext.OrgId
	}

	return proContext
}

func exitOnError(title string, err error) {
	if err != nil {
		log.DefaultLogger.Errorw(title, "error", err)
		os.Exit(1)
	}
}
