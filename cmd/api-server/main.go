package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	cloudresult "github.com/kubeshop/testkube/pkg/cloud/data/result"
	cloudtestresult "github.com/kubeshop/testkube/pkg/cloud/data/testresult"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/version"

	"github.com/kubeshop/testkube/pkg/cloud"
	configmongo "github.com/kubeshop/testkube/pkg/repository/config"
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

	"github.com/kelseyhightower/envconfig"

	kubeclient "github.com/kubeshop/testkube-operator/client"
	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	scriptsclient "github.com/kubeshop/testkube-operator/client/scripts/v2"
	testsclientv1 "github.com/kubeshop/testkube-operator/client/tests"
	testsclientv3 "github.com/kubeshop/testkube-operator/client/tests/v3"
	testsourcesclientv1 "github.com/kubeshop/testkube-operator/client/testsources/v1"
	testsuitesclientv2 "github.com/kubeshop/testkube-operator/client/testsuites/v2"
	apiv1 "github.com/kubeshop/testkube/internal/app/api/v1"
	"github.com/kubeshop/testkube/internal/migrations"
	configmap "github.com/kubeshop/testkube/pkg/config"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/migrator"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/ui"
)

type MongoConfig struct {
	DSN          string `envconfig:"API_MONGO_DSN" default:"mongodb://localhost:27017"`
	DB           string `envconfig:"API_MONGO_DB" default:"testkube"`
	SSLSecretRef string `envconfig:"API_MONGO_SSL_CERT"`
	AllowDiskUse bool   `envconfig:"API_MONGO_ALLOW_DISK_USE"`
}

var Config MongoConfig

var verbose = flag.Bool("v", false, "enable verbosity level")

func init() {
	flag.Parse()
	ui.Verbose = *verbose
	err := envconfig.Process("mongo", &Config)
	ui.PrintOnError("Processing mongo environment config", err)
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

	kubeClient, err := kubeclient.GetClient()
	ui.ExitOnError("Getting kubernetes client", err)

	secretClient, err := secret.NewClient(cfg.TestkubeNamespace)
	ui.ExitOnError("Getting secret client", err)

	// agent
	var grpcClient cloud.TestKubeCloudAPIClient
	mode := common.ModeStandalone
	if cfg.TestkubeCloudAPIKey != "" {
		mode = common.ModeAgent
	}
	if mode == common.ModeAgent {
		grpcConn, err := agent.NewGRPCConnection(ctx, cfg.TestkubeCloudTLSInsecure, cfg.TestkubeCloudURL, log.DefaultLogger)
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
	testsuitesClient := testsuitesclientv2.NewClient(kubeClient, cfg.TestkubeNamespace)
	testsourcesClient := testsourcesclientv1.NewClient(kubeClient, cfg.TestkubeNamespace)

	// DI
	mongoSSLConfig := getMongoSSLConfig(Config, secretClient)
	db, err := storage.GetMongoDatabase(Config.DSN, Config.DB, mongoSSLConfig)
	ui.ExitOnError("Getting mongo database", err)
	var resultsRepository result.Repository
	var testResultsRepository testresult.Repository
	if mode == common.ModeAgent {
		resultsRepository = cloudresult.NewCloudResultRepository(grpcClient, cfg.TestkubeCloudAPIKey)
		testResultsRepository = cloudtestresult.NewCloudRepository(grpcClient, cfg.TestkubeCloudAPIKey)
	} else {
		resultsRepository = result.NewMongoRepository(db, Config.AllowDiskUse)
		testResultsRepository = testresult.NewMongoRepository(db, Config.AllowDiskUse)
	}
	configRepository := configmongo.NewMongoRepository(db)
	configName := fmt.Sprintf("testkube-api-server-config-%s", cfg.TestkubeNamespace)
	if cfg.APIServerConfig != "" {
		configName = cfg.APIServerConfig
	}

	configMapConfig, err := configmap.NewConfigMapConfig(configName, cfg.TestkubeNamespace)
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
		cmConfig.EnableTelemetry = telemetryEnabled
		if cmConfig.ClusterId == "" {
			cmConfig.ClusterId, err = configMapConfig.GetUniqueClusterId(ctx)
		}

		clusterId = cmConfig.ClusterId
		err = configMapConfig.Upsert(ctx, cmConfig)
	}

	log.DefaultLogger.Debugw("Getting unique clusterId", "clusterId", clusterId, "error", err)

	// TODO check if this version exists somewhere in stats (probably could be removed)
	migrations.Migrator.Add(migrations.NewVersion_0_9_2(scriptsClient, testsClientV1, testsClientV3, testsuitesClient))
	if err := runMigrations(); err != nil {
		ui.ExitOnError("Running server migrations", err)
	}

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

	apiVersion := version.Version

	// configure NATS event bus
	nc, err := bus.NewNATSConnection()
	if err != nil {
		log.DefaultLogger.Errorw("error creating NATS connection", "error", err)
	}
	eventBus := bus.NewNATSBus(nc)
	eventsEmitter := event.NewEmitter(eventBus)

	metrics := metrics.NewMetrics()

	templates, err := kubeexecutor.NewTemplatesFromEnv("TESTKUBE_TEMPLATE")
	if err != nil {
		ui.ExitOnError("Creating job templates", err)
	}

	images, err := kubeexecutor.SyncDefaultExecutors(executorsClient, cfg.TestkubeNamespace, cfg.TestkubeDefaultExecutors, cfg.TestkubeReadonlyExecutors)
	if err != nil {
		ui.ExitOnError("Sync default executors", err)
	}

	executor, err := client.NewJobExecutor(
		resultsRepository,
		cfg.TestkubeNamespace,
		images,
		templates,
		cfg.JobServiceAccountName,
		metrics,
		eventsEmitter,
		configMapConfig,
		testsClientV3,
	)
	if err != nil {
		ui.ExitOnError("Creating executor client", err)
	}

	containerTemplates, err := kubeexecutor.NewTemplatesFromEnv("TESTKUBE_CONTAINER_TEMPLATE")
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
		testsuitesClient,
		testsourcesClient,
		secretClient,
		eventsEmitter,
		log.DefaultLogger,
		configMapConfig,
	)

	api := apiv1.NewTestkubeAPI(
		cfg.TestkubeNamespace,
		resultsRepository,
		testResultsRepository,
		testsClientV3,
		executorsClient,
		testsuitesClient,
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
		templates,
		sched,
	)

	isMinioStorage := cfg.LogsStorage == "minio"
	if api.Storage != nil && isMinioStorage && mode != common.ModeAgent {
		bucket := cfg.LogsBucket
		if bucket == "" {
			log.DefaultLogger.Error("LOGS_BUCKET env var is not set")
		} else if _, err := api.Storage.ListBuckets(); err == nil {
			log.DefaultLogger.Info("setting minio as logs storage")
			mongoResultsRepository, ok := resultsRepository.(*result.MongoRepository)
			if ok {
				mongoResultsRepository.OutputRepository = result.NewMinioOutputRepository(api.Storage, mongoResultsRepository.ResultsColl, bucket)
			}
		} else {
			log.DefaultLogger.Infow("minio is not available, using default logs storage", "error", err)
		}
	}

	if mode == common.ModeAgent {
		log.DefaultLogger.Info("starting agent service")

		agentHandle, err := agent.NewAgent(log.DefaultLogger, api.Mux.Handler(), cfg.TestkubeCloudAPIKey, grpcClient)
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
		eventsEmitter.Register(agentHandle)
	}

	api.InitEvents()

	triggerService := triggers.NewService(
		sched,
		clientset,
		testkubeClientset,
		testsuitesClient,
		testsClientV3,
		resultsRepository,
		testResultsRepository,
		triggers.NewMongoLeaseBackend(db),
		log.DefaultLogger,
		configMapConfig,
		triggers.WithHostnameIdentifier(),
	)
	log.DefaultLogger.Info("starting trigger service")
	triggerService.Run(ctx)

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

	if err := g.Wait(); err != nil {
		log.DefaultLogger.Fatalf("Testkube is shutting down: %v", err)
	}
}

// getMongoSSLConfig builds the necessary SSL connection info from the settings in the environment variables
// and the given secret reference
func getMongoSSLConfig(c MongoConfig, secretClient *secret.Client) *storage.MongoSSLConfig {
	if c.SSLSecretRef == "" {
		return nil
	}

	clientCertPath := "/tmp/mongodb.pem"
	rootCAPath := "/tmp/mongodb-root-ca.pem"
	mongoSSLSecret, err := secretClient.Get(Config.SSLSecretRef)
	ui.ExitOnError(fmt.Sprintf("Could not get secret %s for MongoDB connection", c.SSLSecretRef), err)

	var keyFile, caFile, pass string
	var ok bool
	if keyFile, ok = mongoSSLSecret["sslClientCertificateKeyFile"]; !ok {
		ui.Warn("Could not find sslClientCertificateKeyFile in secret %s", c.SSLSecretRef)
	}
	if caFile, ok = mongoSSLSecret["sslCertificateAuthorityFile"]; !ok {
		ui.Warn("Could not find sslCertificateAuthorityFile in secret %s", c.SSLSecretRef)
	}
	if pass, ok = mongoSSLSecret["sslClientCertificateKeyFilePassword"]; !ok {
		ui.Warn("Could not find sslClientCertificateKeyFilePassword in secret %s", c.SSLSecretRef)
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
