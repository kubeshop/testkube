package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

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
	"github.com/kubeshop/testkube/internal/pkg/api"
	configmap "github.com/kubeshop/testkube/internal/pkg/api/config"
	configmongo "github.com/kubeshop/testkube/internal/pkg/api/repository/config"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/storage"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/testresult"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/migrator"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/ui"
)

type MongoConfig struct {
	DSN string `envconfig:"API_MONGO_DSN" default:"mongodb://localhost:27017"`
	DB  string `envconfig:"API_MONGO_DB" default:"testkube"`
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
	results := migrations.Migrator.GetValidMigrations(api.Version, migrator.MigrationTypeServer)
	if len(results) == 0 {
		log.DefaultLogger.Debugw("No migrations available for Testkube", "apiVersion", api.Version)
		return nil
	}

	migrationInfo := []string{}
	for _, migration := range results {
		migrationInfo = append(migrationInfo, fmt.Sprintf("%+v - %s", migration.Version(), migration.Info()))
	}
	log.DefaultLogger.Infow("Available migrations for Testkube", "apiVersion", api.Version, "migrations", migrationInfo)

	return migrations.Migrator.Run(api.Version, migrator.MigrationTypeServer)
}

func main() {

	port := os.Getenv("APISERVER_PORT")
	namespace := "testkube"
	if ns, ok := os.LookupEnv("TESTKUBE_NAMESPACE"); ok {
		namespace = ns
	}

	ln, err := net.Listen("tcp", ":"+port)
	ui.ExitOnError("Checking if port "+port+"is free", err)
	ln.Close()
	log.DefaultLogger.Debugw("TCP Port is available", "port", port)

	// DI
	db, err := storage.GetMongoDataBase(Config.DSN, Config.DB)
	ui.ExitOnError("Getting mongo database", err)

	kubeClient, err := kubeclient.GetClient()
	ui.ExitOnError("Getting kubernetes client", err)

	secretClient, err := secret.NewClient(namespace)
	ui.ExitOnError("Getting secret client", err)

	scriptsClient := scriptsclient.NewClient(kubeClient, namespace)
	testsClientV1 := testsclientv1.NewClient(kubeClient, namespace)
	testsClientV3 := testsclientv3.NewClient(kubeClient, namespace)
	executorsClient := executorsclientv1.NewClient(kubeClient, namespace)
	webhooksClient := executorsclientv1.NewWebhooksClient(kubeClient, namespace)
	testsuitesClient := testsuitesclientv2.NewClient(kubeClient, namespace)
	testsourcesClient := testsourcesclientv1.NewClient(kubeClient, namespace)

	resultsRepository := result.NewMongoRespository(db)
	testResultsRepository := testresult.NewMongoRespository(db)
	configRepository := configmongo.NewMongoRespository(db)
	configName := fmt.Sprintf("testkube-api-server-config-%s", namespace)
	if os.Getenv("APISERVER_CONFIG") != "" {
		configName = os.Getenv("APISERVER_CONFIG")
	}

	configMapConfig, err := configmap.NewConfigMapConfig(configName, namespace)
	ui.ExitOnError("Getting config map config", err)

	ctx := context.Background()
	// try to load from mongo based config first
	telemetryEnabled, err := configMapConfig.GetTelemetryEnabled(ctx)
	if err != nil {
		// fallback to envs in case of failure (no record yet, or other error)
		telemetryEnabled = envs.IsTrue("TESTKUBE_ANALYTICS_ENABLED")
	}

	var clusterId string
	config, err := configMapConfig.Get(ctx)
	if config.ClusterId != "" {
		clusterId = config.ClusterId
	}

	if clusterId == "" {
		config, err = configRepository.Get(ctx)
		config.EnableTelemetry = telemetryEnabled
		if config.ClusterId == "" {
			config.ClusterId, err = configMapConfig.GetUniqueClusterId(ctx)
		}

		clusterId = config.ClusterId
		err = configMapConfig.Upsert(ctx, config)
	}

	log.DefaultLogger.Debugw("Getting uniqe clusterId", "clusterId", clusterId, "error", err)

	// TODO check if this version exists somewhere in stats (probably could be removed)
	migrations.Migrator.Add(migrations.NewVersion_0_9_2(scriptsClient, testsClientV1, testsClientV3, testsuitesClient))
	if err := runMigrations(); err != nil {
		ui.ExitOnError("Running server migrations", err)
	}

	clientset, err := k8sclient.ConnectToK8s()
	if err != nil {
		ui.ExitOnError("Creating k8s clientset", err)
	}

	cfg, err := k8sclient.GetK8sClientConfig()
	if err != nil {
		ui.ExitOnError("Getting k8s client config", err)
	}
	testkubeClientset, err := testkubeclientset.NewForConfig(cfg)
	if err != nil {
		ui.ExitOnError("Creating TestKube Clientset", err)
	}

	apiVersion := api.Version

	// configure NATS event bus
	nc, err := bus.NewNATSConnection()
	if err != nil {
		log.DefaultLogger.Errorw("error creating NATS connection", "error", err)
	}
	eventBus := bus.NewNATSBus(nc)
	eventsEmitter := event.NewEmitter(eventBus)

	api := apiv1.NewTestkubeAPI(
		namespace,
		resultsRepository,
		testResultsRepository,
		testsClientV3,
		executorsClient,
		testsuitesClient,
		secretClient,
		webhooksClient,
		testkubeClientset,
		testsourcesClient,
		configMapConfig,
		clusterId,
		eventsEmitter,
	)

	runner := scheduler.NewScheduler(
		api.Executor,
		resultsRepository,
		testResultsRepository,
		executorsClient,
		testsClientV3,
		testsuitesClient,
		secretClient,
		eventsEmitter,
		log.DefaultLogger,
	)
	triggerService := triggers.NewService(
		runner,
		clientset,
		testkubeClientset,
		testsuitesClient,
		testsClientV3,
		resultsRepository,
		testResultsRepository,
		log.DefaultLogger,
		&api,
	)
	log.DefaultLogger.Info("starting trigger service")
	err = triggerService.Run(ctx)
	if err != nil {
		ui.ExitOnError("Running trigger service", err)
	}

	// telemetry based functions
	api.SendTelemetryStartEvent()
	api.StartTelemetryHeartbeats()

	log.DefaultLogger.Infow(
		"starting Testkube API server",
		"telemetryEnabled", telemetryEnabled,
		"clusterId", clusterId,
		"namespace", namespace,
		"version", apiVersion,
	)

	err = api.Run()
	ui.ExitOnError("Running API Server", err)
}
