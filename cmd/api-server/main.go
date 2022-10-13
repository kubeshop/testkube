package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"golang.org/x/sync/errgroup"

	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/containerexecutor"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	metrics := metrics.NewMetrics()

	executor, err := newExecutorClient(resultsRepository, executorsClient, eventsEmitter, metrics, namespace)
	if err != nil {
		ui.ExitOnError("Creating executor client")
	}

	jobTemplates, err := apiv1.NewJobTemplatesFromEnv("TESTKUBE_TEMPLATE")
	if err != nil {
		ui.ExitOnError("Creating job templates")
	}

	containerExecutor, err := newContainerExecutor(resultsRepository, executorsClient, eventsEmitter, metrics, namespace)
	if err != nil {
		ui.ExitOnError("Creating container executor")
	}

	scheduler := scheduler.NewScheduler(
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
	)

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
		executor,
		containerExecutor,
		metrics,
		jobTemplates,
		scheduler,
	)

	triggerService := triggers.NewService(
		scheduler,
		clientset,
		testkubeClientset,
		testsuitesClient,
		testsClientV3,
		resultsRepository,
		testResultsRepository,
		triggers.NewMongoLeaseBackend(db),
		log.DefaultLogger,
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
		"namespace", namespace,
		"version", apiVersion,
	)

	g.Go(func() error {
		return api.Run(ctx)
	})

	if err := g.Wait(); err != nil {
		log.DefaultLogger.Fatalf("Testkube is shutting down: %v", err)
	}
}

func newContainerExecutor(
	testExecutionResults result.Repository,
	executorsClient executorsclientv1.Interface,
	eventsEmitter *event.Emitter,
	metrics metrics.Metrics,
	namespace string,
) (executor client.Executor, err error) {
	readOnlyExecutors := false
	if value, ok := os.LookupEnv("TESTKUBE_READONLY_EXECUTORS"); ok {
		readOnlyExecutors, err = strconv.ParseBool(value)
		if err != nil {
			return nil, errors.WithMessage(err, "error parsing as bool envvar: TESTKUBE_READONLY_EXECUTORS")
		}
	}

	defaultExecutors := os.Getenv("TESTKUBE_DEFAULT_EXECUTORS")

	initImage, err := loadDefaultExecutors(executorsClient, namespace, defaultExecutors, readOnlyExecutors)
	if err != nil {
		return nil, errors.WithMessage(err, "error loading default executors")
	}

	var jobTemplate string
	jobTemplates, err := apiv1.NewJobTemplatesFromEnv("TESTKUBE_CONTAINER_TEMPLATE")
	if err != nil {
		jobTemplate = ""
	} else {
		jobTemplate = jobTemplates.Job
	}

	return containerexecutor.NewContainerExecutor(testExecutionResults, namespace, initImage, jobTemplate, metrics, eventsEmitter)
}

func newExecutorClient(
	testExecutionResults result.Repository,
	executorsClient executorsclientv1.Interface,
	eventsEmitter *event.Emitter,
	metrics metrics.Metrics,
	namespace string,
) (executor client.Executor, err error) {
	readOnlyExecutors := false
	if value, ok := os.LookupEnv("TESTKUBE_READONLY_EXECUTORS"); ok {
		readOnlyExecutors, err = strconv.ParseBool(value)
		if err != nil {
			return nil, errors.WithMessage(err, "error parsing as bool envvar: TESTKUBE_READONLY_EXECUTORS")
		}
	}

	defaultExecutors := os.Getenv("TESTKUBE_DEFAULT_EXECUTORS")
	initImage, err := loadDefaultExecutors(executorsClient, namespace, defaultExecutors, readOnlyExecutors)
	if err != nil {
		return nil, errors.WithMessage(err, "error loading default executors")
	}

	jobTemplates, err := apiv1.NewJobTemplatesFromEnv("TESTKUBE_TEMPLATE")
	if err != nil {
		return nil, errors.WithMessage(err, "error creating job templates from envvars")
	}

	return client.NewJobExecutor(testExecutionResults, namespace, initImage, jobTemplates.Job, metrics, eventsEmitter)
}

// loadDefaultExecutors loads default executors
func loadDefaultExecutors(executorsClient executorsclientv1.Interface, namespace, data string, readOnlyExecutors bool) (initImage string, err error) {
	var executors []testkube.ExecutorDetails

	if data == "" {
		return "", nil
	}

	dataDecoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(dataDecoded, &executors); err != nil {
		return "", err
	}

	for _, executor := range executors {
		if executor.Executor == nil {
			continue
		}

		if executor.Name == "executor-init" {
			initImage = executor.Executor.Image
			continue
		}

		if readOnlyExecutors {
			continue
		}

		var features []executorv1.Feature
		for _, f := range executor.Executor.Features {
			features = append(features, executorv1.Feature(f))
		}

		obj := &executorv1.Executor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      executor.Name,
				Namespace: namespace,
			},
			Spec: executorv1.ExecutorSpec{
				Types:        executor.Executor.Types,
				ExecutorType: executor.Executor.ExecutorType,
				Image:        executor.Executor.Image,
				Features:     features,
			},
		}

		result, err := executorsClient.Get(executor.Name)
		if err != nil && !k8serrors.IsNotFound(err) {
			return "", err
		}
		if err != nil {
			if _, err = executorsClient.Create(obj); err != nil {
				return "", err
			}
		} else {
			result.Spec = obj.Spec
			if _, err = executorsClient.Update(result); err != nil {
				return "", err
			}
		}
	}

	return initImage, nil
}
