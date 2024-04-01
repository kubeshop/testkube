package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/nats-io/nats.go"

	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	apitclv1 "github.com/kubeshop/testkube/pkg/tcl/apitcl/v1"
	"github.com/kubeshop/testkube/pkg/tcl/checktcl"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/tcl/cloudtcl/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/tcl/repositorytcl/testworkflow"
	"github.com/kubeshop/testkube/pkg/tcl/schedulertcl"

	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	cloudartifacts "github.com/kubeshop/testkube/pkg/cloud/data/artifact"

	domainstorage "github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/slack"

	cloudconfig "github.com/kubeshop/testkube/pkg/cloud/data/config"

	cloudresult "github.com/kubeshop/testkube/pkg/cloud/data/result"
	cloudtestresult "github.com/kubeshop/testkube/pkg/cloud/data/testresult"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	dbmigrations "github.com/kubeshop/testkube/internal/db-migrations"
	parser "github.com/kubeshop/testkube/internal/template"
	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/version"

	"github.com/kubeshop/testkube/pkg/cloud"
	configrepository "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/storage"
	"github.com/kubeshop/testkube/pkg/repository/testresult"

	"golang.org/x/sync/errgroup"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/internal/app/api/debug"
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
	scriptsclient "github.com/kubeshop/testkube-operator/pkg/client/scripts/v2"
	templatesclientv1 "github.com/kubeshop/testkube-operator/pkg/client/templates/v1"
	testexecutionsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testexecutions/v1"
	testsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/tests"
	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	testsourcesclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testsources/v1"
	testsuiteexecutionsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testsuiteexecutions/v1"
	testsuitesclientv2 "github.com/kubeshop/testkube-operator/pkg/client/testsuites/v2"
	testsuitesclientv3 "github.com/kubeshop/testkube-operator/pkg/client/testsuites/v3"
	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/internal/migrations"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/dbmigrator"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/migrator"
	"github.com/kubeshop/testkube/pkg/reconciler"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/ui"
)

var verbose = flag.Bool("v", false, "enable verbosity level")

func init() {
	flag.Parse()
	ui.Verbose = *verbose
}

func runMigrations() (err error) {
	results := migrations.Migrator.GetValidMigrations(version.Version, migrator.MigrationTypeServer)
	if len(results) == 0 {
		log.DefaultLogger.Debugw("No migrations available for Testkube", "apiVersion", version.Version)
		return nil
	}

	var migrationInfo []string
	for _, migration := range results {
		migrationInfo = append(migrationInfo, fmt.Sprintf("%+v - %s", migration.Version(), migration.Info()))
	}
	log.DefaultLogger.Infow("Available migrations for Testkube", "apiVersion", version.Version, "migrations", migrationInfo)

	return migrations.Migrator.Run(version.Version, migrator.MigrationTypeServer)
}

func runMongoMigrations(ctx context.Context, db *mongo.Database, migrationsDir string) error {
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
	cfg, err := config.Get()
	cfg.CleanLegacyVars()
	ui.ExitOnError("error getting application config", err)

	features, err := featureflags.Get()
	ui.ExitOnError("error getting application feature flags", err)

	log.DefaultLogger.Infow("Feature flags configured", "ff", features)

	// Run services within an errgroup to propagate errors between services.
	g, ctx := errgroup.WithContext(context.Background())

	// Cancel the errgroup context on SIGINT and SIGTERM,
	// which shuts everything down gracefully.
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	g.Go(func() error {
		select {
		case <-ctx.Done():
			return nil
		case sig := <-stopSignal:
			go func() {
				<-stopSignal
				os.Exit(137)
			}()
			// Returning an error cancels the errgroup.
			return errors.Errorf("received signal: %v", sig)
		}
	})

	ln, err := net.Listen("tcp", ":"+cfg.APIServerPort)
	ui.ExitOnError("Checking if port "+cfg.APIServerPort+"is free", err)
	_ = ln.Close()
	log.DefaultLogger.Debugw("TCP Port is available", "port", cfg.APIServerPort)

	ln, err = net.Listen("tcp", ":"+cfg.GraphqlPort)
	ui.ExitOnError("Checking if port "+cfg.GraphqlPort+"is free", err)
	_ = ln.Close()
	log.DefaultLogger.Debugw("TCP Port is available", "port", cfg.GraphqlPort)

	kubeClient, err := kubeclient.GetClient()
	ui.ExitOnError("Getting kubernetes client", err)

	secretClient, err := secret.NewClient(cfg.TestkubeNamespace)
	ui.ExitOnError("Getting secret client", err)

	configMapClient, err := configmap.NewClient(cfg.TestkubeNamespace)
	ui.ExitOnError("Getting config map client", err)
	// agent
	var grpcClient cloud.TestKubeCloudAPIClient
	var grpcConn *grpc.ClientConn
	mode := common.ModeStandalone
	if cfg.TestkubeProAPIKey != "" {
		mode = common.ModeAgent
	}
	if mode == common.ModeAgent {
		grpcConn, err = agent.NewGRPCConnection(
			ctx,
			cfg.TestkubeProTLSInsecure,
			cfg.TestkubeProSkipVerify,
			cfg.TestkubeProURL,
			cfg.TestkubeProCertFile,
			cfg.TestkubeProKeyFile,
			cfg.TestkubeProCAFile,
			log.DefaultLogger,
		)
		ui.ExitOnError("error creating gRPC connection", err)
		defer grpcConn.Close()

		grpcClient = cloud.NewTestKubeCloudAPIClient(grpcConn)
	}

	if cfg.EnableDebugServer {
		debugSrv := debug.NewDebugServer(cfg.DebugListenAddr)

		g.Go(func() error {
			log.DefaultLogger.Infof("starting debug pprof server")
			return debugSrv.ListenAndServe()
		})
	}

	// k8s
	scriptsClient := scriptsclient.NewClient(kubeClient, cfg.TestkubeNamespace)
	testsClientV1 := testsclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)
	testsClientV3 := testsclientv3.NewClient(kubeClient, cfg.TestkubeNamespace)
	executorsClient := executorsclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)
	webhooksClient := executorsclientv1.NewWebhooksClient(kubeClient, cfg.TestkubeNamespace)
	testsuitesClientV2 := testsuitesclientv2.NewClient(kubeClient, cfg.TestkubeNamespace)
	testsuitesClientV3 := testsuitesclientv3.NewClient(kubeClient, cfg.TestkubeNamespace)
	testsourcesClient := testsourcesclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)
	testExecutionsClient := testexecutionsclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)
	testsuiteExecutionsClient := testsuiteexecutionsclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)
	templatesClient := templatesclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)

	clientset, err := k8sclient.ConnectToK8s()
	if err != nil {
		ui.ExitOnError("Creating k8s clientset", err)
	}

	k8sCfg, err := k8sclient.GetK8sClientConfig()
	if err != nil {
		ui.ExitOnError("Getting k8s client config", err)
	}
	testkubeClientset, err := testkubeclientset.NewForConfig(k8sCfg)
	if err != nil {
		ui.ExitOnError("Creating TestKube Clientset", err)
	}

	var logGrpcClient logsclient.StreamGetter
	if features.LogsV2 {
		creds, err := newGRPCTransportCredentials(cfg)
		ui.ExitOnError("Getting log server TLS credentials", err)
		logGrpcClient = logsclient.NewGrpcClient(cfg.LogServerGrpcAddress, creds)
	}

	// DI
	var resultsRepository result.Repository
	var testResultsRepository testresult.Repository
	var testWorkflowResultsRepository testworkflow.Repository
	var testWorkflowOutputRepository testworkflow.OutputRepository
	var configRepository configrepository.Repository
	var triggerLeaseBackend triggers.LeaseBackend
	var artifactStorage domainstorage.ArtifactsStorage
	var storageClient domainstorage.Client
	if mode == common.ModeAgent {
		resultsRepository = cloudresult.NewCloudResultRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
		testResultsRepository = cloudtestresult.NewCloudRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
		configRepository = cloudconfig.NewCloudResultRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
		testWorkflowResultsRepository = cloudtestworkflow.NewCloudRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
		testWorkflowOutputRepository = cloudtestworkflow.NewCloudOutputRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
		triggerLeaseBackend = triggers.NewAcquireAlwaysLeaseBackend()
		artifactStorage = cloudartifacts.NewCloudArtifactsStorage(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
	} else {
		mongoSSLConfig := getMongoSSLConfig(cfg, secretClient)
		db, err := storage.GetMongoDatabase(cfg.APIMongoDSN, cfg.APIMongoDB, cfg.APIMongoDBType, cfg.APIMongoAllowTLS, mongoSSLConfig)
		ui.ExitOnError("Getting mongo database", err)
		isDocDb := cfg.APIMongoDBType == storage.TypeDocDB
		mongoResultsRepository := result.NewMongoRepository(db, cfg.APIMongoAllowDiskUse, isDocDb, result.WithFeatureFlags(features), result.WithLogsClient(logGrpcClient))
		resultsRepository = mongoResultsRepository
		testResultsRepository = testresult.NewMongoRepository(db, cfg.APIMongoAllowDiskUse, isDocDb)
		testWorkflowResultsRepository = testworkflow.NewMongoRepository(db, cfg.APIMongoAllowDiskUse)
		configRepository = configrepository.NewMongoRepository(db)
		triggerLeaseBackend = triggers.NewMongoLeaseBackend(db)
		minioClient := newStorageClient(cfg)
		if err = minioClient.Connect(); err != nil {
			ui.ExitOnError("Connecting to minio", err)
		}
		if expErr := minioClient.SetExpirationPolicy(cfg.StorageExpiration); expErr != nil {
			log.DefaultLogger.Errorw("Error setting expiration policy", "error", expErr)
		}
		storageClient = minioClient
		testWorkflowOutputRepository = testworkflow.NewMinioOutputRepository(storageClient, cfg.LogsBucket)
		artifactStorage = minio.NewMinIOArtifactClient(storageClient)
		// init storage
		isMinioStorage := cfg.LogsStorage == "minio"
		if isMinioStorage {
			bucket := cfg.LogsBucket
			if bucket == "" {
				log.DefaultLogger.Error("LOGS_BUCKET env var is not set")
			} else if ok, err := storageClient.IsConnectionPossible(ctx); ok && (err == nil) {
				log.DefaultLogger.Info("setting minio as logs storage")
				mongoResultsRepository.OutputRepository = result.NewMinioOutputRepository(storageClient, mongoResultsRepository.ResultsColl, bucket)
			} else {
				log.DefaultLogger.Infow("minio is not available, using default logs storage", "error", err)
			}
		}

		// Run DB migrations
		if !cfg.DisableMongoMigrations {
			err := runMongoMigrations(ctx, db, filepath.Join(cfg.TestkubeConfigDir, "db-migrations"))
			if err != nil {
				log.DefaultLogger.Warnf("failed to apply MongoDB migrations: %v", err)
			}
		}
	}

	configName := fmt.Sprintf("testkube-api-server-config-%s", cfg.TestkubeNamespace)
	if cfg.APIServerConfig != "" {
		configName = cfg.APIServerConfig
	}

	configMapConfig, err := configrepository.NewConfigMapConfig(configName, cfg.TestkubeNamespace)
	ui.ExitOnError("Getting config map config", err)

	// try to load from mongo based config first
	telemetryEnabled, err := configMapConfig.GetTelemetryEnabled(ctx)
	if err != nil {
		// fallback to envs in case of failure (no record yet, or other error)
		telemetryEnabled = cfg.TestkubeAnalyticsEnabled
	}

	var clusterId string
	cmConfig, err := configMapConfig.Get(ctx)
	if cmConfig.ClusterId != "" {
		clusterId = cmConfig.ClusterId
	}

	if clusterId == "" {
		cmConfig, err = configRepository.Get(ctx)
		if err != nil {
			log.DefaultLogger.Warnw("error fetching config ConfigMap", "error", err)
		}
		cmConfig.EnableTelemetry = telemetryEnabled
		if cmConfig.ClusterId == "" {
			cmConfig.ClusterId, err = configMapConfig.GetUniqueClusterId(ctx)
			if err != nil {
				log.DefaultLogger.Warnw("error getting unique clusterId", "error", err)
			}
		}

		clusterId = cmConfig.ClusterId
		_, err = configMapConfig.Upsert(ctx, cmConfig)
		if err != nil {
			log.DefaultLogger.Warn("error upserting config ConfigMap", "error", err)
		}

	}

	log.DefaultLogger.Debugw("Getting unique clusterId", "clusterId", clusterId, "error", err)

	// TODO check if this version exists somewhere in stats (probably could be removed)
	migrations.Migrator.Add(migrations.NewVersion_0_9_2(scriptsClient, testsClientV1, testsClientV3, testsuitesClientV2))
	if err := runMigrations(); err != nil {
		ui.ExitOnError("Running server migrations", err)
	}

	apiVersion := version.Version

	envs := make(map[string]string)
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}

		envs[pair[0]] += pair[1]
	}

	nc, err := newNATSConnection(cfg)
	if err != nil {
		ui.ExitOnError("Creating NATS connection", err)
	}
	eventBus := bus.NewNATSBus(nc)
	eventsEmitter := event.NewEmitter(eventBus, cfg.TestkubeClusterName, envs)

	var logsStream logsclient.Stream

	if features.LogsV2 {
		logsStream, err = logsclient.NewNatsLogStream(nc.Conn)
		if err != nil {
			ui.ExitOnError("Creating logs streaming client", err)
		}
	}

	metrics := metrics.NewMetrics()

	defaultExecutors, err := parseDefaultExecutors(cfg)
	if err != nil {
		ui.ExitOnError("Parsing default executors", err)
	}

	images, err := kubeexecutor.SyncDefaultExecutors(executorsClient, cfg.TestkubeNamespace, defaultExecutors, cfg.TestkubeReadonlyExecutors)
	if err != nil {
		ui.ExitOnError("Sync default executors", err)
	}

	jobTemplates, err := parser.ParseJobTemplates(cfg)
	if err != nil {
		ui.ExitOnError("Creating job templates", err)
	}

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
	}

	// Check Pro/Enterprise subscription
	var subscriptionChecker checktcl.SubscriptionChecker
	if mode == common.ModeAgent {
		subscriptionChecker, err = checktcl.NewSubscriptionChecker(ctx, proContext, grpcClient, grpcConn)
		ui.ExitOnError("Failed creating subscription checker", err)
	}

	serviceAccountNames := map[string]string{
		cfg.TestkubeNamespace: cfg.JobServiceAccountName,
	}

	// Pro edition only (tcl protected code)
	if cfg.TestkubeExecutionNamespaces != "" {
		err = subscriptionChecker.IsActiveOrgPlanEnterpriseForFeature("execution namespace")
		ui.ExitOnError("Subscription checking", err)

		serviceAccountNames = schedulertcl.GetServiceAccountNamesFromConfig(serviceAccountNames, cfg.TestkubeExecutionNamespaces)
	}

	executor, err := client.NewJobExecutor(
		resultsRepository,
		images,
		jobTemplates,
		serviceAccountNames,
		metrics,
		eventsEmitter,
		configMapConfig,
		testsClientV3,
		clientset,
		testExecutionsClient,
		templatesClient,
		cfg.TestkubeRegistry,
		cfg.TestkubePodStartTimeout,
		clusterId,
		cfg.TestkubeDashboardURI,
		"http://"+cfg.APIServerFullname+":"+cfg.APIServerPort,
		cfg.NatsURI,
		cfg.Debug,
		logsStream,
		features,
	)
	if err != nil {
		ui.ExitOnError("Creating executor client", err)
	}

	containerTemplates, err := parser.ParseContainerTemplates(cfg)
	if err != nil {
		ui.ExitOnError("Creating container job templates", err)
	}

	inspectorStorages := []imageinspector.Storage{imageinspector.NewMemoryStorage()}
	if cfg.EnableImageDataPersistentCache {
		configmapStorage := imageinspector.NewConfigMapStorage(configMapClient, cfg.ImageDataPersistentCacheKey, true)
		_ = configmapStorage.CopyTo(context.Background(), inspectorStorages[0].(imageinspector.StorageTransfer))
		inspectorStorages = append(inspectorStorages, configmapStorage)
	}
	inspector := imageinspector.NewInspector(
		cfg.TestkubeRegistry,
		imageinspector.NewSkopeoFetcher(),
		imageinspector.NewSecretFetcher(secretClient),
		inspectorStorages...,
	)

	containerExecutor, err := containerexecutor.NewContainerExecutor(
		resultsRepository,
		images,
		containerTemplates,
		inspector,
		serviceAccountNames,
		metrics,
		eventsEmitter,
		configMapConfig,
		executorsClient,
		testsClientV3,
		testExecutionsClient,
		templatesClient,
		cfg.TestkubeRegistry,
		cfg.TestkubePodStartTimeout,
		clusterId,
		cfg.TestkubeDashboardURI,
		"http://"+cfg.APIServerFullname+":"+cfg.APIServerPort,
		cfg.NatsURI,
		cfg.Debug,
		logsStream,
		features,
	)
	if err != nil {
		ui.ExitOnError("Creating container executor", err)
	}

	sched := scheduler.NewScheduler(
		metrics,
		executor,
		containerExecutor,
		resultsRepository,
		testResultsRepository,
		executorsClient,
		testsClientV3,
		testsuitesClientV3,
		testsourcesClient,
		secretClient,
		eventsEmitter,
		log.DefaultLogger,
		configMapConfig,
		configMapClient,
		testsuiteExecutionsClient,
		eventBus,
		cfg.TestkubeDashboardURI,
		features,
		logsStream,
		cfg.TestkubeNamespace,
		cfg.TestkubeProTLSSecret,
	)
	if mode == common.ModeAgent {
		sched.WithSubscriptionChecker(subscriptionChecker)
	}

	slackLoader, err := newSlackLoader(cfg, envs)
	if err != nil {
		ui.ExitOnError("Creating slack loader", err)
	}

	api := apiv1.NewTestkubeAPI(
		cfg.TestkubeNamespace,
		resultsRepository,
		testResultsRepository,
		testsClientV3,
		executorsClient,
		testsuitesClientV3,
		secretClient,
		webhooksClient,
		clientset,
		testkubeClientset,
		testsourcesClient,
		configMapConfig,
		clusterId,
		eventsEmitter,
		executor,
		containerExecutor,
		metrics,
		sched,
		slackLoader,
		storageClient,
		cfg.GraphqlPort,
		artifactStorage,
		templatesClient,
		cfg.CDEventsTarget,
		cfg.TestkubeDashboardURI,
		cfg.TestkubeHelmchartVersion,
		mode,
		eventBus,
		cfg.EnableSecretsEndpoint,
		features,
		logsStream,
		logGrpcClient,
		cfg.DisableSecretCreation,
		subscriptionChecker,
		serviceAccountNames,
	)

	// Apply Pro server enhancements
	apiPro := apitclv1.NewApiTCL(
		api,
		&proContext,
		kubeClient,
		inspector,
		testWorkflowResultsRepository,
		testWorkflowOutputRepository,
		"http://"+cfg.APIServerFullname+":"+cfg.APIServerPort,
		configMapConfig,
	)
	apiPro.AppendRoutes()

	if mode == common.ModeAgent {
		log.DefaultLogger.Info("starting agent service")
		api.WithProContext(&proContext)
		agentHandle, err := agent.NewAgent(
			log.DefaultLogger,
			api.Mux.Handler(),
			grpcClient,
			api.GetLogsStream,
			apiPro.GetTestWorkflowNotificationsStream,
			clusterId,
			cfg.TestkubeClusterName,
			envs,
			features,
			proContext,
		)
		if err != nil {
			ui.ExitOnError("Starting agent", err)
		}
		g.Go(func() error {
			err = agentHandle.Run(ctx)
			if err != nil {
				ui.ExitOnError("Running agent", err)
			}
			return nil
		})
		eventsEmitter.Loader.Register(agentHandle)
	}

	api.InitEvents()
	if !cfg.DisableTestTriggers {
		triggerService := triggers.NewService(
			sched,
			clientset,
			testkubeClientset,
			testsuitesClientV3,
			testsClientV3,
			resultsRepository,
			testResultsRepository,
			triggerLeaseBackend,
			log.DefaultLogger,
			configMapConfig,
			executorsClient,
			executor,
			eventBus,
			metrics,
			triggers.WithHostnameIdentifier(),
			triggers.WithTestkubeNamespace(cfg.TestkubeNamespace),
			triggers.WithWatcherNamespaces(cfg.TestkubeWatcherNamespaces),
			triggers.WithDisableSecretCreation(cfg.DisableSecretCreation),
		)
		log.DefaultLogger.Info("starting trigger service")
		triggerService.Run(ctx)
	} else {
		log.DefaultLogger.Info("test triggers are disabled")
	}

	if !cfg.DisableReconciler {
		reconcilerClient := reconciler.NewClient(clientset,
			resultsRepository,
			testResultsRepository,
			executorsClient,
			log.DefaultLogger)
		g.Go(func() error {
			return reconcilerClient.Run(ctx)
		})
	} else {
		log.DefaultLogger.Info("reconclier is disabled")
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

func parseDefaultExecutors(cfg *config.Config) (executors []testkube.ExecutorDetails, err error) {
	rawExecutors, err := parser.LoadConfigFromStringOrFile(
		cfg.TestkubeDefaultExecutors,
		cfg.TestkubeConfigDir,
		"executors.json",
		"executors",
	)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal([]byte(rawExecutors), &executors); err != nil {
		return nil, err
	}

	enabledExecutors, err := parser.LoadConfigFromStringOrFile(
		cfg.TestkubeEnabledExecutors,
		cfg.TestkubeConfigDir,
		"enabledExecutors",
		"enabled executors",
	)
	if err != nil {
		return nil, err
	}

	specifiedExecutors := make(map[string]struct{})
	if enabledExecutors != "" {
		for _, executor := range strings.Split(enabledExecutors, ",") {
			if strings.TrimSpace(executor) == "" {
				continue
			}

			specifiedExecutors[strings.TrimSpace(executor)] = struct{}{}
		}

		for i := len(executors) - 1; i >= 0; i-- {
			if _, ok := specifiedExecutors[executors[i].Name]; !ok {
				executors = append(executors[:i], executors[i+1:]...)
			}
		}
	}

	return executors, nil
}

func newNATSConnection(cfg *config.Config) (*nats.EncodedConn, error) {
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

func newStorageClient(cfg *config.Config) *minio.Client {
	opts := minio.GetTLSOptions(cfg.StorageSSL, cfg.StorageSkipVerify, cfg.StorageCertFile, cfg.StorageKeyFile, cfg.StorageCAFile)
	return minio.NewClient(
		cfg.StorageEndpoint,
		cfg.StorageAccessKeyID,
		cfg.StorageSecretAccessKey,
		cfg.StorageRegion,
		cfg.StorageToken,
		cfg.StorageBucket,
		opts...,
	)
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

// getMongoSSLConfig builds the necessary SSL connection info from the settings in the environment variables
// and the given secret reference
func getMongoSSLConfig(cfg *config.Config, secretClient *secret.Client) *storage.MongoSSLConfig {
	if cfg.APIMongoSSLCert == "" {
		return nil
	}

	clientCertPath := "/tmp/mongodb.pem"
	rootCAPath := "/tmp/mongodb-root-ca.pem"
	mongoSSLSecret, err := secretClient.Get(cfg.APIMongoSSLCert)
	ui.ExitOnError(fmt.Sprintf("Could not get secret %s for MongoDB connection", cfg.APIMongoSSLCert), err)

	var keyFile, caFile, pass string
	var ok bool
	if keyFile, ok = mongoSSLSecret[cfg.APIMongoSSLClientFileKey]; !ok {
		ui.Warn("Could not find sslClientCertificateKeyFile with key %s in secret %s", cfg.APIMongoSSLClientFileKey, cfg.APIMongoSSLCert)
	}
	if caFile, ok = mongoSSLSecret[cfg.APIMongoSSLCAFileKey]; !ok {
		ui.Warn("Could not find sslCertificateAuthorityFile with key %s in secret %s", cfg.APIMongoSSLCAFileKey, cfg.APIMongoSSLCert)
	}
	if pass, ok = mongoSSLSecret[cfg.APIMongoSSLClientFilePass]; !ok {
		ui.Warn("Could not find sslClientCertificateKeyFilePassword with key %s in secret %s", cfg.APIMongoSSLClientFilePass, cfg.APIMongoSSLCert)
	}

	err = os.WriteFile(clientCertPath, []byte(keyFile), 0644)
	ui.ExitOnError(fmt.Sprintf("Could not place mongodb certificate key file: %s", err))

	err = os.WriteFile(rootCAPath, []byte(caFile), 0644)
	ui.ExitOnError(fmt.Sprintf("Could not place mongodb ssl ca file: %s", err))

	return &storage.MongoSSLConfig{
		SSLClientCertificateKeyFile:         clientCertPath,
		SSLClientCertificateKeyFilePassword: pass,
		SSLCertificateAuthoritiyFile:        rootCAPath,
	}
}

func newGRPCTransportCredentials(cfg *config.Config) (credentials.TransportCredentials, error) {
	return logsclient.GetGrpcTransportCredentials(logsclient.GrpcConnectionConfig{
		Secure:     cfg.LogServerSecure,
		SkipVerify: cfg.LogServerSkipVerify,
		CertFile:   cfg.LogServerCertFile,
		KeyFile:    cfg.LogServerKeyFile,
		CAFile:     cfg.LogServerCAFile,
	})
}
