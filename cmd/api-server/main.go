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
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/pkg/cache"

	"github.com/nats-io/nats.go"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	cloudtestworkflow "github.com/kubeshop/testkube/pkg/cloud/data/testworkflow"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	testworkflow2 "github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/secretmanager"
	"github.com/kubeshop/testkube/pkg/tcl/checktcl"
	"github.com/kubeshop/testkube/pkg/tcl/controlplanetcl"
	"github.com/kubeshop/testkube/pkg/tcl/schedulertcl"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/presets"

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
	"github.com/kubeshop/testkube/pkg/repository/sequence"
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
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/internal/migrations"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/dbmigrator"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/migrator"
	"github.com/kubeshop/testkube/pkg/reconciler"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

func init() {
	flag.Parse()
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
	exitOnError("error getting application config", err)

	md := metadata.Pairs("api-key", cfg.TestkubeProAPIKey, "runner-id", cfg.TestkubeProRunnerId)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	features, err := featureflags.Get()
	exitOnError("error getting application feature flags", err)

	logger := log.DefaultLogger.With("apiVersion", version.Version)

	logger.Infow("Feature flags configured", "ff", features)

	// Run services within an errgroup to propagate errors between services.
	g, ctx := errgroup.WithContext(ctx)

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
	exitOnError("Checking if port "+cfg.APIServerPort+"is free", err)
	_ = ln.Close()
	log.DefaultLogger.Debugw("TCP Port is available", "port", cfg.APIServerPort)

	ln, err = net.Listen("tcp", ":"+cfg.GraphqlPort)
	exitOnError("Checking if port "+cfg.GraphqlPort+"is free", err)
	_ = ln.Close()
	log.DefaultLogger.Debugw("TCP Port is available", "port", cfg.GraphqlPort)

	kubeClient, err := kubeclient.GetClient()
	exitOnError("Getting kubernetes client", err)

	secretClient, err := secret.NewClient(cfg.TestkubeNamespace)
	exitOnError("Getting secret client", err)

	configMapClient, err := configmap.NewClient(cfg.TestkubeNamespace)
	exitOnError("Getting config map client", err)
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
			cfg.TestkubeProCAFile, //nolint
			log.DefaultLogger,
		)
		exitOnError("error creating gRPC connection", err)
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
	testWorkflowsClient := testworkflowsclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)
	testWorkflowTemplatesClient := testworkflowsclientv1.NewTestWorkflowTemplatesClient(kubeClient, cfg.TestkubeNamespace)
	testWorkflowExecutionsClient := testworkflowsclientv1.NewTestWorkflowExecutionsClient(kubeClient, cfg.TestkubeNamespace)
	templatesClient := templatesclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)

	clientset, err := k8sclient.ConnectToK8s()
	if err != nil {
		exitOnError("Creating k8s clientset", err)
	}

	k8sCfg, err := k8sclient.GetK8sClientConfig()
	if err != nil {
		exitOnError("Getting k8s client config", err)
	}
	testkubeClientset, err := testkubeclientset.NewForConfig(k8sCfg)
	if err != nil {
		exitOnError("Creating TestKube Clientset", err)
	}

	var logGrpcClient logsclient.StreamGetter
	if features.LogsV2 {
		creds, err := newGRPCTransportCredentials(cfg)
		exitOnError("Getting log server TLS credentials", err)
		logGrpcClient = logsclient.NewGrpcClient(cfg.LogServerGrpcAddress, creds)
	}

	// DI
	var resultsRepository result.Repository
	var testResultsRepository testresult.Repository
	var testWorkflowResultsRepository testworkflow2.Repository
	var testWorkflowOutputRepository testworkflow2.OutputRepository
	var configRepository configrepository.Repository
	var triggerLeaseBackend triggers.LeaseBackend
	var artifactStorage domainstorage.ArtifactsStorage
	var storageClient domainstorage.Client
	if mode == common.ModeAgent {
		resultsRepository = cloudresult.NewCloudResultRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey, cfg.TestkubeProRunnerId)
		testResultsRepository = cloudtestresult.NewCloudRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey, cfg.TestkubeProRunnerId)
		configRepository = cloudconfig.NewCloudResultRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey, cfg.TestkubeProRunnerId)
		// Pro edition only (tcl protected code)
		testWorkflowResultsRepository = cloudtestworkflow.NewCloudRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey, cfg.TestkubeProRunnerId)
		var opts []cloudtestworkflow.Option
		if cfg.StorageSkipVerify {
			opts = append(opts, cloudtestworkflow.WithSkipVerify())
		}
		testWorkflowOutputRepository = cloudtestworkflow.NewCloudOutputRepository(grpcClient, grpcConn, cfg.TestkubeProAPIKey, cfg.TestkubeProRunnerId, opts...)
		triggerLeaseBackend = triggers.NewAcquireAlwaysLeaseBackend()
		artifactStorage = cloudartifacts.NewCloudArtifactsStorage(grpcClient, grpcConn, cfg.TestkubeProAPIKey, cfg.TestkubeProRunnerId)
	} else {
		mongoSSLConfig := getMongoSSLConfig(cfg, secretClient)
		db, err := storage.GetMongoDatabase(cfg.APIMongoDSN, cfg.APIMongoDB, cfg.APIMongoDBType, cfg.APIMongoAllowTLS, mongoSSLConfig)
		exitOnError("Getting mongo database", err)
		isDocDb := cfg.APIMongoDBType == storage.TypeDocDB
		sequenceRepository := sequence.NewMongoRepository(db)
		mongoResultsRepository := result.NewMongoRepository(db, cfg.APIMongoAllowDiskUse, isDocDb, result.WithFeatureFlags(features),
			result.WithLogsClient(logGrpcClient), result.WithMongoRepositorySequence(sequenceRepository))
		resultsRepository = mongoResultsRepository
		testResultsRepository = testresult.NewMongoRepository(db, cfg.APIMongoAllowDiskUse, isDocDb,
			testresult.WithMongoRepositorySequence(sequenceRepository))
		testWorkflowResultsRepository = testworkflow2.NewMongoRepository(db, cfg.APIMongoAllowDiskUse,
			testworkflow2.WithMongoRepositorySequence(sequenceRepository))
		configRepository = configrepository.NewMongoRepository(db)
		triggerLeaseBackend = triggers.NewMongoLeaseBackend(db)
		minioClient := newStorageClient(cfg)
		if err = minioClient.Connect(); err != nil {
			exitOnError("Connecting to minio", err)
		}
		if expErr := minioClient.SetExpirationPolicy(cfg.StorageExpiration); expErr != nil {
			log.DefaultLogger.Errorw("Error setting expiration policy", "error", expErr)
		}
		storageClient = minioClient
		testWorkflowOutputRepository = testworkflow2.NewMinioOutputRepository(storageClient, db.Collection(testworkflow2.CollectionName), cfg.LogsBucket)
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
	exitOnError("Getting config map config", err)

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
		exitOnError("Running server migrations", err)
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

	nc, err := newNATSEncodedConnection(cfg)
	if err != nil {
		exitOnError("Creating NATS connection", err)
	}

	eventBus := bus.NewNATSBus(nc)
	if cfg.Trace {
		eventBus.TraceEvents()
	}

	eventsEmitter := event.NewEmitter(eventBus, cfg.TestkubeClusterName, envs)

	var logsStream logsclient.Stream

	if features.LogsV2 {
		logsStream, err = logsclient.NewNatsLogStream(nc.Conn)
		if err != nil {
			exitOnError("Creating logs streaming client", err)
		}
	}

	metrics := metrics.NewMetrics()

	defaultExecutors, err := parseDefaultExecutors(cfg)
	if err != nil {
		exitOnError("Parsing default executors", err)
	}

	images, err := kubeexecutor.SyncDefaultExecutors(executorsClient, cfg.TestkubeNamespace, defaultExecutors, cfg.TestkubeReadonlyExecutors)
	if err != nil {
		exitOnError("Sync default executors", err)
	}

	jobTemplates, err := parser.ParseJobTemplates(cfg)
	if err != nil {
		exitOnError("Creating job templates", err)
	}

	proContext := newProContext(ctx, cfg, grpcClient)
	proContext.ClusterId = clusterId

	// Check Pro/Enterprise subscription
	var subscriptionChecker checktcl.SubscriptionChecker
	if mode == common.ModeAgent {
		subscriptionChecker, err = checktcl.NewSubscriptionChecker(ctx, proContext, grpcClient, grpcConn)
		exitOnError("Failed creating subscription checker", err)

		// Load environment/org details based on token grpc call
		environment, err := controlplanetcl.GetEnvironment(ctx, proContext, grpcClient, grpcConn)
		warnOnError("Getting environment details from control plane", err)
		proContext.EnvID = environment.Id
		proContext.EnvName = environment.Name
		proContext.EnvSlug = environment.Slug
		proContext.OrgID = environment.OrganizationId
		proContext.OrgName = environment.OrganizationName
		proContext.OrgSlug = environment.OrganizationSlug
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
		cfg.TestkubeDefaultStorageClassName,
		cfg.WhitelistedContainers,
	)
	if err != nil {
		exitOnError("Creating executor client", err)
	}

	containerTemplates, err := parser.ParseContainerTemplates(cfg)
	if err != nil {
		exitOnError("Creating container job templates", err)
	}

	inspectorStorages := []imageinspector.Storage{imageinspector.NewMemoryStorage()}
	if cfg.EnableImageDataPersistentCache {
		configmapStorage := imageinspector.NewConfigMapStorage(configMapClient, cfg.ImageDataPersistentCacheKey, true)
		_ = configmapStorage.CopyTo(context.Background(), inspectorStorages[0].(imageinspector.StorageTransfer))
		inspectorStorages = append(inspectorStorages, configmapStorage)
	}
	inspector := imageinspector.NewInspector(
		cfg.TestkubeRegistry,
		imageinspector.NewCraneFetcher(),
		imageinspector.NewSecretFetcher(secretClient, cache.NewInMemoryCache[*corev1.Secret](), imageinspector.WithSecretCacheTTL(cfg.TestkubeImageCredentialsCacheTTL)),
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
		cfg.TestkubeDefaultStorageClassName,
		cfg.WhitelistedContainers,
		cfg.TestkubeImageCredentialsCacheTTL,
	)
	if err != nil {
		exitOnError("Creating container executor", err)
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
		cfg.TestkubeProRunnerCustomCASecret,
	)
	if mode == common.ModeAgent {
		sched.WithSubscriptionChecker(subscriptionChecker)
	}

	slackLoader, err := newSlackLoader(cfg, envs)
	if err != nil {
		exitOnError("Creating slack loader", err)
	}

	testWorkflowProcessor := presets.NewOpenSource(inspector)
	if mode == common.ModeAgent {
		testWorkflowProcessor = presets.NewPro(inspector)
	}
	testWorkflowExecutor := testworkflowexecutor.New(
		eventsEmitter,
		clientset,
		testWorkflowResultsRepository,
		testWorkflowOutputRepository,
		testWorkflowTemplatesClient,
		testWorkflowProcessor,
		configMapConfig,
		testWorkflowExecutionsClient,
		testWorkflowsClient,
		metrics,
		serviceAccountNames,
		cfg.GlobalWorkflowTemplateName,
		cfg.TestkubeNamespace,
		"http://"+cfg.APIServerFullname+":"+cfg.APIServerPort,
		cfg.TestkubeRegistry,
		cfg.EnableImageDataPersistentCache,
		cfg.ImageDataPersistentCacheKey,
		cfg.TestkubeDashboardURI,
		clusterId,
		proContext.RunnerId,
	)

	go testWorkflowExecutor.Recover(context.Background())

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

	api := apiv1.NewTestkubeAPI(
		cfg.TestkubeNamespace,
		resultsRepository,
		testResultsRepository,
		testWorkflowResultsRepository,
		testWorkflowOutputRepository,
		testsClientV3,
		executorsClient,
		testsuitesClientV3,
		secretClient,
		secretManager,
		webhooksClient,
		clientset,
		testkubeClientset,
		testsourcesClient,
		testWorkflowsClient,
		testWorkflowTemplatesClient,
		configMapConfig,
		clusterId,
		eventsEmitter,
		executor,
		containerExecutor,
		testWorkflowExecutor,
		metrics,
		sched,
		slackLoader,
		storageClient,
		cfg.GraphqlPort,
		artifactStorage,
		templatesClient,
		cfg.TestkubeDashboardURI,
		cfg.TestkubeHelmchartVersion,
		mode,
		eventBus,
		secretConfig,
		features,
		logsStream,
		logGrpcClient,
		subscriptionChecker,
		serviceAccountNames,
	)

	if mode == common.ModeAgent {
		log.DefaultLogger.Info("starting agent service")
		api.WithProContext(&proContext)
		agentHandle, err := agent.NewAgent(
			log.DefaultLogger,
			api.Mux.Handler(),
			grpcClient,
			api.GetLogsStream,
			api.GetTestWorkflowNotificationsStream,
			clusterId,
			cfg.TestkubeClusterName,
			envs,
			features,
			proContext,
		)
		if err != nil {
			exitOnError("Starting agent", err)
		}
		g.Go(func() error {
			err = agentHandle.Run(ctx)
			if err != nil {
				exitOnError("Running agent", err)
			}
			return nil
		})
		eventsEmitter.Loader.Register(agentHandle)
	}

	api.Init(cfg.CDEventsTarget, cfg.EnableK8sEvents)
	if !cfg.DisableTestTriggers {
		triggerService := triggers.NewService(
			sched,
			clientset,
			testkubeClientset,
			testsuitesClientV3,
			testsClientV3,
			testWorkflowsClient,
			resultsRepository,
			testResultsRepository,
			triggerLeaseBackend,
			log.DefaultLogger,
			configMapConfig,
			executorsClient,
			executor,
			eventBus,
			metrics,
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
	exitOnError(fmt.Sprintf("Could not get secret %s for MongoDB connection", cfg.APIMongoSSLCert), err)

	var keyFile, caFile, pass string
	var ok bool
	if keyFile, ok = mongoSSLSecret[cfg.APIMongoSSLClientFileKey]; !ok {
		log.DefaultLogger.Warnf("Could not find sslClientCertificateKeyFile with key %s in secret %s", cfg.APIMongoSSLClientFileKey, cfg.APIMongoSSLCert)
	}
	if caFile, ok = mongoSSLSecret[cfg.APIMongoSSLCAFileKey]; !ok {
		log.DefaultLogger.Warnf("Could not find sslCertificateAuthorityFile with key %s in secret %s", cfg.APIMongoSSLCAFileKey, cfg.APIMongoSSLCert)
	}
	if pass, ok = mongoSSLSecret[cfg.APIMongoSSLClientFilePass]; !ok {
		log.DefaultLogger.Warnf("Could not find sslClientCertificateKeyFilePassword with key %s in secret %s", cfg.APIMongoSSLClientFilePass, cfg.APIMongoSSLCert)
	}

	err = os.WriteFile(clientCertPath, []byte(keyFile), 0644)
	exitOnError("Could not place mongodb certificate key file", err)

	err = os.WriteFile(rootCAPath, []byte(caFile), 0644)
	exitOnError("Could not place mongodb ssl ca file: %s", err)

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

func newProContext(ctx context.Context, cfg *config.Config, grpcClient cloud.TestKubeCloudAPIClient) config.ProContext {
	proContext := config.ProContext{
		APIKey:                           cfg.TestkubeProAPIKey,
		RunnerId:                         cfg.TestkubeProRunnerId,
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

	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	proContextResponse, err := grpcClient.GetProContext(ctx, &emptypb.Empty{})
	if err != nil {
		log.DefaultLogger.Warnf("cannot fetch pro-context from cloud: %s", err)
		return proContext
	}

	if proContext.EnvID == "" {
		proContext.EnvID = proContextResponse.EnvId
	}

	if proContext.OrgID == "" {
		proContext.OrgID = proContextResponse.OrgId
	}

	return proContext
}

func exitOnError(title string, err error) {
	if err != nil {
		log.DefaultLogger.Errorw(title, "error", err)
		os.Exit(1)
	}
}

func warnOnError(title string, err error) {
	if err != nil {
		log.DefaultLogger.Errorw(title, "error", err)
	}
}
