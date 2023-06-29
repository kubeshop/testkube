package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"google.golang.org/grpc"

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
	"github.com/kubeshop/testkube/pkg/version"

	"github.com/kubeshop/testkube/pkg/cloud"
	configrepository "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/storage"
	"github.com/kubeshop/testkube/pkg/repository/testresult"

	"golang.org/x/sync/errgroup"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/agent"
	kubeexecutor "github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/containerexecutor"

	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/scheduler"

	testkubeclientset "github.com/kubeshop/testkube-operator/pkg/clientset/versioned"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/triggers"

	kubeclient "github.com/kubeshop/testkube-operator/client"
	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	scriptsclient "github.com/kubeshop/testkube-operator/client/scripts/v2"
	testsclientv1 "github.com/kubeshop/testkube-operator/client/tests"
	testsclientv3 "github.com/kubeshop/testkube-operator/client/tests/v3"
	testsourcesclientv1 "github.com/kubeshop/testkube-operator/client/testsources/v1"
	testsuitesclientv2 "github.com/kubeshop/testkube-operator/client/testsuites/v2"
	testsuitesclientv3 "github.com/kubeshop/testkube-operator/client/testsuites/v3"
	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/internal/migrations"
	"github.com/kubeshop/testkube/pkg/configmap"
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

func main() {
	cfg, err := config.Get()
	ui.ExitOnError("error getting application config", err)
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
	if cfg.TestkubeCloudAPIKey != "" {
		mode = common.ModeAgent
	}
	if mode == common.ModeAgent {
		grpcConn, err = agent.NewGRPCConnection(ctx, cfg.TestkubeCloudTLSInsecure, cfg.TestkubeCloudURL, log.DefaultLogger)
		ui.ExitOnError("error creating gRPC connection", err)
		defer grpcConn.Close()

		grpcClient = cloud.NewTestKubeCloudAPIClient(grpcConn)
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

	clientset, err := k8sclient.ConnectToK8s()
	if err != nil {
		ui.ExitOnError("Creating k8s clientset", err)
	}

	dynamicClientSet, err := k8sclient.ConnectToK8sDynamic()
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

	// DI
	var resultsRepository result.Repository
	var testResultsRepository testresult.Repository
	var configRepository configrepository.Repository
	var triggerLeaseBackend triggers.LeaseBackend
	var artifactStorage domainstorage.ArtifactsStorage
	var storageClient domainstorage.Client
	if mode == common.ModeAgent {
		resultsRepository = cloudresult.NewCloudResultRepository(grpcClient, grpcConn, cfg.TestkubeCloudAPIKey)
		testResultsRepository = cloudtestresult.NewCloudRepository(grpcClient, grpcConn, cfg.TestkubeCloudAPIKey)
		configRepository = cloudconfig.NewCloudResultRepository(grpcClient, grpcConn, cfg.TestkubeCloudAPIKey)
		triggerLeaseBackend = triggers.NewAcquireAlwaysLeaseBackend()
		artifactStorage = cloudartifacts.NewCloudArtifactsStorage(grpcClient, grpcConn, cfg.TestkubeCloudAPIKey)
	} else {
		mongoSSLConfig := getMongoSSLConfig(cfg, secretClient)
		db, err := storage.GetMongoDatabase(cfg.APIMongoDSN, cfg.APIMongoDB, cfg.APIMongoDBType, cfg.APIMongoAllowTLS, mongoSSLConfig)
		ui.ExitOnError("Getting mongo database", err)
		mongoResultsRepository := result.NewMongoRepository(db, cfg.APIMongoAllowDiskUse)
		resultsRepository = mongoResultsRepository
		testResultsRepository = testresult.NewMongoRepository(db, cfg.APIMongoAllowDiskUse)
		configRepository = configrepository.NewMongoRepository(db)
		triggerLeaseBackend = triggers.NewMongoLeaseBackend(db)
		minioClient := minio.NewClient(
			cfg.StorageEndpoint,
			cfg.StorageAccessKeyID,
			cfg.StorageSecretAccessKey,
			cfg.StorageRegion,
			cfg.StorageToken,
			cfg.StorageBucket,
			cfg.StorageSSL,
		)
		if err = minioClient.Connect(); err != nil {
			ui.ExitOnError("Connecting to minio", err)
		}
		if expErr := minioClient.SetExpirationPolicy(cfg.StorageExpiration); expErr != nil {
			log.DefaultLogger.Errorw("Error setting expiration policy", "error", expErr)
		}
		storageClient = minioClient
		artifactStorage = minio.NewMinIOArtifactClient(storageClient)
		// init storage
		isMinioStorage := cfg.LogsStorage == "minio"
		if isMinioStorage {
			bucket := cfg.LogsBucket
			if bucket == "" {
				log.DefaultLogger.Error("LOGS_BUCKET env var is not set")
			} else if _, err := storageClient.ListBuckets(ctx); err == nil {
				log.DefaultLogger.Info("setting minio as logs storage")
				mongoResultsRepository.OutputRepository = result.NewMinioOutputRepository(storageClient, mongoResultsRepository.ResultsColl, bucket)
			} else {
				log.DefaultLogger.Infow("minio is not available, using default logs storage", "error", err)
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

	// configure NATS event bus
	nc, err := bus.NewNATSConnection(cfg.NatsURI)
	if err != nil {
		log.DefaultLogger.Errorw("error creating NATS connection", "error", err)
	}
	eventBus := bus.NewNATSBus(nc)
	eventsEmitter := event.NewEmitter(eventBus, cfg.TestkubeClusterName)

	metrics := metrics.NewMetrics()

	defaultExecutors, err := parseDefaultExecutors(cfg)
	if err != nil {
		ui.ExitOnError("Parsing default executors", err)
	}

	images, err := kubeexecutor.SyncDefaultExecutors(executorsClient, cfg.TestkubeNamespace, defaultExecutors, cfg.TestkubeReadonlyExecutors)
	if err != nil {
		ui.ExitOnError("Sync default executors", err)
	}

	jobTemplate, err := parseJobTemplate(cfg)
	if err != nil {
		ui.ExitOnError("Creating job templates", err)
	}

	executor, err := client.NewJobExecutor(
		resultsRepository,
		cfg.TestkubeNamespace,
		images,
		jobTemplate,
		cfg.JobServiceAccountName,
		metrics,
		eventsEmitter,
		configMapConfig,
		testsClientV3,
		clientset,
		cfg.TestkubeRegistry,
		cfg.TestkubePodStartTimeout,
		clusterId,
	)
	if err != nil {
		ui.ExitOnError("Creating executor client", err)
	}

	containerTemplates, err := parseContainerTemplates(cfg)
	if err != nil {
		ui.ExitOnError("Creating container job templates", err)
	}

	containerExecutor, err := containerexecutor.NewContainerExecutor(
		resultsRepository,
		cfg.TestkubeNamespace,
		images,
		containerTemplates,
		cfg.JobServiceAccountName,
		metrics,
		eventsEmitter,
		configMapConfig,
		executorsClient,
		testsClientV3,
		cfg.TestkubeRegistry,
		cfg.TestkubePodStartTimeout,
		clusterId,
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
	)

	slackLoader, err := newSlackLoader(cfg)
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
		jobTemplate,
		sched,
		slackLoader,
		storageClient,
		cfg.GraphqlPort,
		artifactStorage,
		cfg.CDEventsTarget,
		cfg.TestkubeDashboardURI,
	)

	if mode == common.ModeAgent {
		log.DefaultLogger.Info("starting agent service")

		agentHandle, err := agent.NewAgent(
			log.DefaultLogger,
			api.Mux.Handler(),
			cfg.TestkubeCloudAPIKey,
			grpcClient,
			cfg.TestkubeCloudWorkerCount,
			cfg.TestkubeCloudLogStreamWorkerCount,
			api.GetLogsStream,
			clusterId,
			cfg.TestkubeClusterName,
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
			dynamicClientSet,
			testkubeClientset,
			testsuitesClientV3,
			testsClientV3,
			resultsRepository,
			testResultsRepository,
			triggerLeaseBackend,
			log.DefaultLogger,
			configMapConfig,
			executorsClient,
			triggers.WithHostnameIdentifier(),
			triggers.WithTestkubeNamespace(cfg.TestkubeNamespace),
			triggers.WithWatcherNamespaces(cfg.TestkubeWatcherNamespaces),
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
			log.DefaultLogger,
			cfg.TestkubeNamespace)
		g.Go(func() error {
			return reconcilerClient.Run(ctx)
		})
	} else {
		log.DefaultLogger.Info("reconclier is disabled")
	}

	// telemetry based functions
	api.SendTelemetryStartEvent(ctx)
	api.StartTelemetryHeartbeats(ctx)

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

func parseJobTemplate(cfg *config.Config) (template string, err error) {
	template, err = loadFromBase64StringOrFile(
		cfg.TestkubeTemplateJob,
		cfg.TestkubeConfigDir,
		"job-template.yml",
		"job template",
	)
	if err != nil {
		return "", err
	}

	return template, nil
}

func parseContainerTemplates(cfg *config.Config) (t kubeexecutor.Templates, err error) {
	t.Job, err = loadFromBase64StringOrFile(
		cfg.TestkubeContainerTemplateJob,
		cfg.TestkubeConfigDir,
		"job-container-template.yml",
		"job container template",
	)
	if err != nil {
		return t, err
	}

	t.Scraper, err = loadFromBase64StringOrFile(
		cfg.TestkubeContainerTemplateScraper,
		cfg.TestkubeConfigDir,
		"job-scraper-template.yml",
		"job scraper template",
	)
	if err != nil {
		return t, err
	}

	t.PVC, err = loadFromBase64StringOrFile(
		cfg.TestkubeContainerTemplatePVC,
		cfg.TestkubeConfigDir,
		"pvc-container-template.yml",
		"pvc container template",
	)
	if err != nil {
		return t, err
	}

	return t, nil
}

func parseDefaultExecutors(cfg *config.Config) (executors []testkube.ExecutorDetails, err error) {
	rawExecutors, err := loadFromBase64StringOrFile(
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

	return executors, nil
}

func newSlackLoader(cfg *config.Config) (*slack.SlackLoader, error) {
	slackTemplate, err := loadFromBase64StringOrFile(
		cfg.SlackTemplate,
		cfg.TestkubeConfigDir,
		"slack-template.json",
		"slack template",
	)
	if err != nil {
		return nil, err
	}

	slackConfig, err := loadFromBase64StringOrFile(cfg.SlackConfig, cfg.TestkubeConfigDir, "slack-config.json", "slack config")
	if err != nil {
		return nil, err
	}

	return slack.NewSlackLoader(slackTemplate, slackConfig, cfg.TestkubeClusterName, testkube.AllEventTypes), nil
}

func loadFromBase64StringOrFile(base64Val string, configDir, filename, configType string) (raw string, err error) {
	var data []byte

	if base64Val != "" {
		data, err = base64.StdEncoding.DecodeString(base64Val)
		if err != nil {
			return "", errors.Wrapf(err, "error decoding %s from base64", configType)
		}
		raw = string(data)
		log.DefaultLogger.Infof("parsed %s from env var", configType)
	} else if f, err := os.Open(filepath.Join(configDir, filename)); err == nil {
		data, err = io.ReadAll(f)
		if err != nil {
			return "", errors.Wrapf(err, "error reading file %s from config dir %s", filename, configDir)
		}
		raw = string(data)
		log.DefaultLogger.Infof("loaded %s from file %s", configType, filepath.Join(configDir, filename))
	} else {
		log.DefaultLogger.Infof("no %s config found", configType)
	}

	return raw, nil
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
