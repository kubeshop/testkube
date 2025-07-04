package commons

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	testsuitesclientv3 "github.com/kubeshop/testkube-operator/pkg/client/testsuites/v3"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	mongomigrations "github.com/kubeshop/testkube/internal/db-migrations"
	parser "github.com/kubeshop/testkube/internal/template"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cache"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/cronjob"
	postgresmigrations "github.com/kubeshop/testkube/pkg/database/postgres/migrations"
	"github.com/kubeshop/testkube/pkg/dbmigrator"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/event/kind/slack"
	kubeexecutor "github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/storage"
	"github.com/kubeshop/testkube/pkg/secret"
	domainstorage "github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
	"github.com/kubeshop/testkube/pkg/workerpool"
)

func ExitOnError(title string, err error) {
	if err != nil {
		log.DefaultLogger.Errorw(title, "error", err)
		os.Exit(1)
	}
}

// General

func GetEnvironmentVariables() map[string]string {
	list := os.Environ()
	envs := make(map[string]string, len(list))
	for _, env := range list {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}

		envs[pair[0]] += pair[1]
	}
	return envs
}

func HandleCancelSignal(ctx context.Context) func() error {
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, syscall.SIGINT, syscall.SIGTERM)
	return func() error {
		select {
		case <-ctx.Done():
			return nil
		case sig := <-stopSignal:
			go func() {
				<-stopSignal
				os.Exit(137)
			}()
			// Returning an error cancels the errgroup.
			return fmt.Errorf("received signal: %v", sig)
		}
	}
}

// Configuration

func MustGetConfig() *config.Config {
	cfg, err := config.Get()
	ExitOnError("error getting application config", err)
	return cfg
}

func MustGetFeatureFlags() featureflags.FeatureFlags {
	features, err := featureflags.Get()
	ExitOnError("error getting application feature flags", err)
	log.DefaultLogger.Infow("Feature flags configured", "ff", features)
	return features
}

func MustFreePort(port int) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	ExitOnError(fmt.Sprintf("Checking if port %d is free", port), err)
	_ = ln.Close()
	log.DefaultLogger.Debugw("TCP Port is available", "port", port)
}

func MustGetConfigMapConfig(ctx context.Context, name string, namespace string, defaultTelemetryEnabled bool) *configRepo.ConfigMapConfig {
	if name == "" {
		name = fmt.Sprintf("testkube-api-server-config-%s", namespace)
	}
	configMapConfig, err := configRepo.NewConfigMapConfig(name, namespace)
	ExitOnError("Getting config map config", err)

	// Load the initial data
	err = configMapConfig.Load(ctx, defaultTelemetryEnabled)
	if err != nil {
		log.DefaultLogger.Warn("error upserting config ConfigMap", "error", err)
	}
	return configMapConfig
}

func MustGetMinioClient(cfg *config.Config) domainstorage.Client {
	opts := minio.GetTLSOptions(cfg.StorageSSL, cfg.StorageSkipVerify, cfg.StorageCertFile, cfg.StorageKeyFile, cfg.StorageCAFile)
	minioClient := minio.NewClient(
		cfg.StorageEndpoint,
		cfg.StorageAccessKeyID,
		cfg.StorageSecretAccessKey,
		cfg.StorageRegion,
		cfg.StorageToken,
		cfg.StorageBucket,
		opts...,
	)
	err := minioClient.Connect()
	ExitOnError("Connecting to minio", err)
	if expErr := minioClient.SetExpirationPolicy(cfg.StorageExpiration); expErr != nil {
		log.DefaultLogger.Errorw("Error setting expiration policy", "error", expErr)
	}
	return minioClient
}

func runMongoMigrations(ctx context.Context, db *mongo.Database) error {
	migrationsCollectionName := "__migrations"
	activeMigrations, err := dbmigrator.GetDbMigrationsFromFs(mongomigrations.MongoMigrationsFs)
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

func MustGetMongoDatabase(ctx context.Context, cfg *config.Config, secretClient secret.Interface, migrate bool) *mongo.Database {
	mongoSSLConfig := getMongoSSLConfig(cfg, secretClient)
	db, err := storage.GetMongoDatabase(cfg.APIMongoDSN, cfg.APIMongoDB, cfg.APIMongoDBType, cfg.APIMongoAllowTLS, mongoSSLConfig)
	ExitOnError("Getting mongo database", err)
	if migrate {
		if err = runMongoMigrations(ctx, db); err != nil {
			log.DefaultLogger.Warnf("failed to apply MongoDB migrations: %v", err)
		}
	}
	return db
}

// getMongoSSLConfig builds the necessary SSL connection info from the settings in the environment variables
// and the given secret reference
func getMongoSSLConfig(cfg *config.Config, secretClient secret.Interface) *storage.MongoSSLConfig {
	if cfg.APIMongoSSLCert == "" {
		return nil
	}

	clientCertPath := "/tmp/mongodb.pem"
	rootCAPath := "/tmp/mongodb-root-ca.pem"
	mongoSSLSecret, err := secretClient.Get(cfg.APIMongoSSLCert)
	ExitOnError(fmt.Sprintf("Could not get secret %s for MongoDB connection", cfg.APIMongoSSLCert), err)

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
	ExitOnError("Could not place mongodb certificate key file", err)

	err = os.WriteFile(rootCAPath, []byte(caFile), 0644)
	ExitOnError("Could not place mongodb ssl ca file: %s", err)

	return &storage.MongoSSLConfig{
		SSLClientCertificateKeyFile:         clientCertPath,
		SSLClientCertificateKeyFilePassword: pass,
		SSLCertificateAuthoritiyFile:        rootCAPath,
	}
}

func MustGetPostgresDatabase(ctx context.Context, cfg *config.Config, migrate bool) *pgxpool.Pool {
	if migrate {
		db, err := sql.Open("postgres", cfg.APIPostgresDSN)
		ExitOnError("Getting Postgres database db", err)

		if err := runPostgresMigrations(ctx, db); err != nil {
			log.DefaultLogger.Warnf("failed to apply Postgres migrations: %v", err)
		}

		db.Close()
	}

	// Connect to PostgreSQL
	pool, err := pgxpool.New(context.Background(), cfg.APIPostgresDSN)
	ExitOnError("Getting Postgres database pool", err)

	return pool
}

func runPostgresMigrations(ctx context.Context, db *sql.DB) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, postgresmigrations.Fs)
	if err != nil {
		return errors.Wrap(err, "failed to plan Postgres migrations")
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to apply Postgres migrations")
	}

	if len(results) == 0 {
		log.DefaultLogger.Info("No Postgres migrations to apply.")
	} else {
		log.DefaultLogger.Info(fmt.Sprintf("Applied Postgres migrations with results %v", results))
	}

	return nil
}

// Actions

func ReadDefaultExecutors(cfg *config.Config) (executors []testkube.ExecutorDetails, images kubeexecutor.Images, err error) {
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

func ReadProContext(ctx context.Context, cfg *config.Config, grpcClient cloud.TestKubeCloudAPIClient) config.ProContext {
	proContext := config.ProContext{
		APIKey:                              cfg.ControlPlaneConfig.TestkubeProAPIKey,
		URL:                                 cfg.ControlPlaneConfig.TestkubeProURL,
		TLSInsecure:                         cfg.ControlPlaneConfig.TestkubeProTLSInsecure,
		SkipVerify:                          cfg.ControlPlaneConfig.TestkubeProSkipVerify,
		EnvID:                               cfg.ControlPlaneConfig.TestkubeProEnvID,
		EnvSlug:                             cfg.ControlPlaneConfig.TestkubeProEnvID,
		EnvName:                             cfg.ControlPlaneConfig.TestkubeProEnvID,
		OrgID:                               cfg.ControlPlaneConfig.TestkubeProOrgID,
		OrgSlug:                             cfg.ControlPlaneConfig.TestkubeProOrgID,
		OrgName:                             cfg.ControlPlaneConfig.TestkubeProOrgID,
		ConnectionTimeout:                   cfg.ControlPlaneConfig.TestkubeProConnectionTimeout,
		WorkerCount:                         cfg.TestkubeProWorkerCount,
		LogStreamWorkerCount:                cfg.TestkubeProLogStreamWorkerCount,
		Migrate:                             cfg.TestkubeProMigrate,
		DashboardURI:                        cfg.TestkubeDashboardURI,
		NewArchitecture:                     grpcClient == nil,
		CloudStorage:                        grpcClient == nil,
		CloudStorageSupportedInControlPlane: grpcClient == nil,
	}
	proContext.Agent.ID = cfg.ControlPlaneConfig.TestkubeProAgentID
	proContext.Agent.Name = cfg.ControlPlaneConfig.TestkubeProAgentID

	cloudUiUrl := os.Getenv("TESTKUBE_PRO_UI_URL")
	if proContext.DashboardURI == "" && cloudUiUrl != "" {
		proContext.DashboardURI = cloudUiUrl
	}
	proContext.DashboardURI = strings.TrimRight(proContext.DashboardURI, "/")

	if cfg.TestkubeProAPIKey == "" || grpcClient == nil {
		return proContext
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		"api-key":         proContext.APIKey,
		"organization-id": proContext.OrgID,
		"agent-id":        proContext.Agent.ID,
	}))
	defer cancel()
	foundProContext, err := grpcClient.GetProContext(ctx, &emptypb.Empty{})
	if err != nil {
		log.DefaultLogger.Warnf("cannot fetch pro-context from cloud: %s", err)
		return proContext
	}

	if proContext.EnvID == "" {
		proContext.EnvID = foundProContext.EnvId
	}

	if proContext.Agent.ID == "" && strings.HasPrefix(proContext.APIKey, "tkcagnt_") {
		proContext.Agent.ID = strings.Replace(foundProContext.EnvId, "tkcenv_", "tkcroot_", 1)
	}

	if proContext.OrgID == "" {
		proContext.OrgID = foundProContext.OrgId
	}

	if foundProContext.OrgName != "" {
		proContext.OrgName = foundProContext.OrgName
	}

	if foundProContext.OrgSlug != "" {
		proContext.OrgSlug = foundProContext.OrgSlug
	}

	foundDashboardUrl := strings.TrimRight(foundProContext.PublicDashboardUrl, "/")
	if foundDashboardUrl != "" {
		proContext.DashboardURI = foundDashboardUrl
	}

	if foundProContext.Agent != nil && foundProContext.Agent.Id != "" {
		proContext.Agent.ID = foundProContext.Agent.Id
		proContext.Agent.Name = foundProContext.Agent.Name
		proContext.Agent.Type = foundProContext.Agent.Type
		proContext.Agent.Labels = foundProContext.Agent.Labels
		proContext.Agent.Disabled = foundProContext.Agent.Disabled
		proContext.Agent.Environments = common.MapSlice(foundProContext.Agent.Environments, func(env *cloud.ProContextEnvironment) config.ProContextAgentEnvironment {
			return config.ProContextAgentEnvironment{
				ID:   env.Id,
				Slug: env.Slug,
				Name: env.Name,
			}
		})

		for _, env := range foundProContext.Agent.Environments {
			if env.Id == proContext.EnvID {
				proContext.EnvName = env.Name
				proContext.EnvSlug = env.Slug
			}
		}
	}

	if cfg.FeatureNewArchitecture && capabilities.Enabled(foundProContext.Capabilities, capabilities.CapabilityNewArchitecture) {
		proContext.NewArchitecture = true
	}

	if capabilities.Enabled(foundProContext.Capabilities, capabilities.CapabilityCloudStorage) {
		proContext.CloudStorageSupportedInControlPlane = true
		if cfg.FeatureCloudStorage {
			proContext.CloudStorage = true
		}
	}

	return proContext
}

func MustCreateSlackLoader(cfg *config.Config, envs map[string]string) *slack.SlackLoader {
	slackTemplate, err := parser.LoadConfigFromStringOrFile(
		cfg.SlackTemplate,
		cfg.TestkubeConfigDir,
		"slack-template.json",
		"slack template",
	)
	ExitOnError("Creating slack loader", err)

	slackConfig, err := parser.LoadConfigFromStringOrFile(cfg.SlackConfig, cfg.TestkubeConfigDir, "slack-config.json", "slack config")
	ExitOnError("Creating slack loader", err)

	return slack.NewSlackLoader(slackTemplate, slackConfig, cfg.TestkubeClusterName, cfg.TestkubeDashboardURI,
		testkube.AllEventTypes, envs)
}

func MustCreateNATSConnection(cfg *config.Config) *nats.EncodedConn {
	// if embedded NATS server is enabled, we'll replace connection with one to the embedded server
	if cfg.NatsEmbedded {
		_, nc, err := event.ServerWithConnection(cfg.NatsEmbeddedStoreDir)
		ExitOnError("Creating NATS connection", err)

		log.DefaultLogger.Info("Started embedded NATS server")

		conn, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
		ExitOnError("Creating NATS connection", err)
		return conn
	}

	conn, err := bus.NewNATSEncodedConnection(bus.ConnectionConfig{
		NatsURI:            cfg.NatsURI,
		NatsSecure:         cfg.NatsSecure,
		NatsSkipVerify:     cfg.NatsSkipVerify,
		NatsCertFile:       cfg.NatsCertFile,
		NatsKeyFile:        cfg.NatsKeyFile,
		NatsCAFile:         cfg.NatsCAFile,
		NatsConnectTimeout: cfg.NatsConnectTimeout,
	})
	ExitOnError("Creating NATS connection", err)
	return conn
}

// Components

func CreateImageInspector(cfg *config.ImageInspectorConfig, configMapClient configmap.Interface, secretClient secret.Interface) imageinspector.Inspector {
	inspectorStorages := []imageinspector.Storage{imageinspector.NewMemoryStorage()}
	if cfg.EnableImageDataPersistentCache {
		configmapStorage := imageinspector.NewConfigMapStorage(configMapClient, cfg.ImageDataPersistentCacheKey, true)
		_ = configmapStorage.CopyTo(context.Background(), inspectorStorages[0].(imageinspector.StorageTransfer))
		inspectorStorages = append(inspectorStorages, configmapStorage)
	}
	return imageinspector.NewInspector(
		cfg.TestkubeRegistry,
		imageinspector.NewCraneFetcher(),
		imageinspector.NewSecretFetcher(secretClient, cache.NewInMemoryCache[*corev1.Secret](), imageinspector.WithSecretCacheTTL(cfg.TestkubeImageCredentialsCacheTTL)),
		inspectorStorages...,
	)
}

func CreateCronJobScheduler(cfg *config.Config,
	kubeClient kubeclient.Client,
	testWorkflowClient testworkflowclient.TestWorkflowClient,
	testWorkflowTemplateClient testworkflowtemplateclient.TestWorkflowTemplateClient,
	testWorkflowExecutor testworkflowexecutor.TestWorkflowExecutor,
	deprecatedClients DeprecatedClients,
	executeTestFn workerpool.ExecuteFn[testkube.Test, testkube.ExecutionRequest, testkube.Execution],
	executeTestSuiteFn workerpool.ExecuteFn[testkube.TestSuite, testkube.TestSuiteExecutionRequest, testkube.TestSuiteExecution],
	logger *zap.SugaredLogger,
	kubeConfig *rest.Config,
	proContext *config.ProContext) cronjob.Interface {
	enableCronJobs := cfg.EnableCronJobs
	if enableCronJobs == "" {
		var err error
		enableCronJobs, err = parser.LoadConfigFromFile(
			cfg.TestkubeConfigDir,
			"enable-cron-jobs",
			"enable cron jobs",
		)
		ExitOnError("Creating cron job scheduler config loading", err)
	}

	if enableCronJobs == "" {
		return nil
	}

	result, err := strconv.ParseBool(enableCronJobs)
	ExitOnError("Creating cron job scheduler config parsing", err)

	if !result {
		return nil
	}

	var testClient testsclientv3.Interface
	var testSuiteClient testsuitesclientv3.Interface
	var testRESTClient testsclientv3.RESTInterface
	var testSuiteRESTClient testsuitesclientv3.RESTInterface
	if deprecatedClients != nil {
		testClient = deprecatedClients.Tests()
		testSuiteClient = deprecatedClients.TestSuites()
		testRESTClient, err = testsclientv3.NewRESTClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)
		ExitOnError("Creating cron job scheduler test rest client", err)
		testSuiteRESTClient, err = testsuitesclientv3.NewRESTClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)
		ExitOnError("Creating cron job scheduler test suite rest client", err)
	}

	scheduler := cronjob.New(testWorkflowClient,
		testWorkflowTemplateClient,
		testWorkflowExecutor,
		logger,
		cronjob.WithProContext(proContext),
		cronjob.WithTestClient(testClient),
		cronjob.WithTestSuiteClient(testSuiteClient),
		cronjob.WithExecuteTestFn(executeTestFn),
		cronjob.WithExecuteTestSuiteFn(executeTestSuiteFn),
		cronjob.WithTestRESTClient(testRESTClient),
		cronjob.WithTestSuiteRESTClient(testSuiteRESTClient),
	)

	return scheduler
}
