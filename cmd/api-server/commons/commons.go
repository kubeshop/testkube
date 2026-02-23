package commons

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	mongomigrations "github.com/kubeshop/testkube/internal/db-migrations"
	parser "github.com/kubeshop/testkube/internal/template"
	"github.com/kubeshop/testkube/pkg/cache"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/configmap"
	postgresmigrations "github.com/kubeshop/testkube/pkg/database/postgres/migrations"
	"github.com/kubeshop/testkube/pkg/dbmigrator"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/log"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/storage"
	"github.com/kubeshop/testkube/pkg/secret"
	domainstorage "github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

func ExitOnError(title string, err error) {
	if err != nil {
		log.DefaultLogger.Errorw(title, "error", err)
		os.Exit(1)
	}
}

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

func MustFreePort(port int) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	ExitOnError(fmt.Sprintf("checking if port %d is free", port), err)
	_ = ln.Close()
	log.DefaultLogger.Debugw("TCP Port is available", "port", port)
}

func MustGetConfigMapConfig(ctx context.Context, name string, namespace string, defaultTelemetryEnabled bool) *configRepo.ConfigMapConfig {
	if name == "" {
		name = fmt.Sprintf("testkube-api-server-config-%s", namespace)
	}
	configMapConfig, err := configRepo.NewConfigMapConfig(name, namespace)
	ExitOnError("getting config map config", err)

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
		log.DefaultLogger.Info("no MongoDB migrations to apply.")
	} else {
		log.DefaultLogger.Info(fmt.Sprintf("applying MongoDB migrations: %d rollbacks and %d ups.", len(plan.Downs), len(plan.Ups)))
	}
	err = dbMigrator.Apply(ctx)
	return errors.Wrap(err, "failed to apply MongoDB migrations")
}

func MustGetMongoDatabase(ctx context.Context, cfg *config.Config, secretClient secret.Interface, migrate bool) *mongo.Database {
	mongoSSLConfig := getMongoSSLConfig(cfg, secretClient)
	db, err := storage.GetMongoDatabase(cfg.APIMongoDSN, cfg.APIMongoDB, cfg.APIMongoDBType, cfg.APIMongoAllowTLS, mongoSSLConfig)
	ExitOnError("getting mongo database", err)
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
		log.DefaultLogger.Warnf("could not find sslClientCertificateKeyFile with key %s in secret %s", cfg.APIMongoSSLClientFileKey, cfg.APIMongoSSLCert)
	}
	if caFile, ok = mongoSSLSecret[cfg.APIMongoSSLCAFileKey]; !ok {
		log.DefaultLogger.Warnf("could not find sslCertificateAuthorityFile with key %s in secret %s", cfg.APIMongoSSLCAFileKey, cfg.APIMongoSSLCert)
	}
	if pass, ok = mongoSSLSecret[cfg.APIMongoSSLClientFilePass]; !ok {
		log.DefaultLogger.Warnf("could not find sslClientCertificateKeyFilePassword with key %s in secret %s", cfg.APIMongoSSLClientFilePass, cfg.APIMongoSSLCert)
	}

	err = os.WriteFile(clientCertPath, []byte(keyFile), 0644)
	ExitOnError("could not place mongodb certificate key file", err)

	err = os.WriteFile(rootCAPath, []byte(caFile), 0644)
	ExitOnError("could not place mongodb ssl ca file: %s", err)

	return &storage.MongoSSLConfig{
		SSLClientCertificateKeyFile:         clientCertPath,
		SSLClientCertificateKeyFilePassword: pass,
		SSLCertificateAuthoritiyFile:        rootCAPath,
	}
}

func MustGetPostgresDatabase(ctx context.Context, cfg *config.Config, migrate bool) *pgxpool.Pool {
	// Connect to PostgreSQL
	pool, err := pgxpool.New(context.Background(), cfg.APIPostgresDSN)
	ExitOnError("Getting Postgres database", err)

	if migrate {
		db := stdlib.OpenDBFromPool(pool)
		if err := runPostgresMigrations(ctx, db); err != nil {
			log.DefaultLogger.Warnf("failed to apply Postgres migrations: %v", err)
		}
	}

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
func retryPostgresMigrations(ctx context.Context, db *sql.DB) {
	delay := 1 * time.Second
	maxDelay := 30 * time.Second
	maxAttempts := 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := runPostgresMigrations(ctx, db); err == nil {
			return
		} else {
			log.DefaultLogger.Warnw("failed to apply Postgres migrations; will retry", "error", err, "backoff", delay, "attempt", attempt+1)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		if delay < maxDelay {
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
	log.DefaultLogger.Errorw("failed to apply Postgres migrations after max retries", "attempts", maxAttempts)
}

func ReadProContext(ctx context.Context, cfg *config.Config, grpcClient cloud.TestKubeCloudAPIClient) (config.ProContext, error) {
	proContext := config.ProContext{
		APIKey:                              cfg.TestkubeProAPIKey,
		URL:                                 cfg.TestkubeProURL,
		TLSInsecure:                         cfg.TestkubeProTLSInsecure,
		SkipVerify:                          cfg.TestkubeProSkipVerify,
		EnvID:                               cfg.TestkubeProEnvID,
		EnvSlug:                             cfg.TestkubeProEnvID,
		EnvName:                             cfg.TestkubeProEnvID,
		OrgID:                               cfg.TestkubeProOrgID,
		OrgSlug:                             cfg.TestkubeProOrgID,
		OrgName:                             cfg.TestkubeProOrgID,
		ConnectionTimeout:                   cfg.TestkubeProConnectionTimeout,
		WorkerCount:                         cfg.TestkubeProWorkerCount,
		Migrate:                             cfg.TestkubeProMigrate,
		DashboardURI:                        cfg.TestkubeDashboardURI,
		CloudStorage:                        false,
		CloudStorageSupportedInControlPlane: false,
	}
	proContext.Agent.ID = cfg.TestkubeProAgentID
	proContext.Agent.Name = cfg.TestkubeProAgentID

	cloudUiUrl := os.Getenv("TESTKUBE_PRO_UI_URL")
	if proContext.DashboardURI == "" && cloudUiUrl != "" {
		proContext.DashboardURI = cloudUiUrl
	}
	proContext.DashboardURI = strings.TrimRight(proContext.DashboardURI, "/")

	ctx, cancel := context.WithTimeout(ctx, time.Second*3)

	if proContext.APIKey != "" {
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
			"api-key":         proContext.APIKey,
			"organization-id": proContext.OrgID,
			"agent-id":        proContext.Agent.ID,
		}))
	}
	defer cancel()
	foundProContext, err := grpcClient.GetProContext(ctx, &emptypb.Empty{})
	if err != nil {
		log.DefaultLogger.Warnf("cannot fetch pro-context from cloud: %s", err)
		return proContext, fmt.Errorf("cannot get pro context: %v", err)
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
		proContext.Agent.Labels = foundProContext.Agent.Labels
		proContext.Agent.Disabled = foundProContext.Agent.Disabled
		proContext.Agent.IsSuperAgent = foundProContext.Agent.IsSuperAgent
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

	if capabilities.Enabled(foundProContext.Capabilities, capabilities.CapabilityCloudStorage) {
		proContext.CloudStorageSupportedInControlPlane = true
		if cfg.FeatureCloudStorage {
			proContext.CloudStorage = true
		}
	}

	if capabilities.Enabled(foundProContext.Capabilities, capabilities.CapabilitySourceOfTruth) {
		proContext.HasSourceOfTruthCapability = true
	}

	return proContext, nil
}

func MustCreateNATSConnection(cfg *config.Config) *nats.EncodedConn { //nolint:staticcheck
	// if embedded NATS server is enabled, we'll replace connection with one to the embedded server
	if cfg.NatsEmbedded {
		_, nc, err := event.ServerWithConnection(cfg.NatsEmbeddedStoreDir)
		ExitOnError("creating NATS connection", err)

		log.DefaultLogger.Info("started embedded NATS server")

		conn, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER) //nolint:staticcheck
		ExitOnError("creating NATS connection", err)
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
	ExitOnError("creating NATS connection", err)
	return conn
}

// Components

func trimAndFilterRegistries(csv string) []string {
	var result []string
	for _, r := range strings.Split(csv, ",") {
		if r = strings.TrimSpace(r); r != "" {
			result = append(result, r)
		}
	}
	return result
}

func CreateImageInspector(cfg *config.ImageInspectorConfig, configMapClient configmap.Interface, secretClient secret.Interface) imageinspector.Inspector {
	inspectorStorages := []imageinspector.Storage{imageinspector.NewMemoryStorage()}
	if cfg.EnableImageDataPersistentCache {
		configmapStorage := imageinspector.NewConfigMapStorage(configMapClient, cfg.ImageDataPersistentCacheKey, true)
		_ = configmapStorage.CopyTo(context.Background(), inspectorStorages[0].(imageinspector.StorageTransfer))
		inspectorStorages = append(inspectorStorages, configmapStorage)
	}
	return imageinspector.NewInspector(
		cfg.TestkubeRegistry,
		imageinspector.NewCraneFetcher(trimAndFilterRegistries(cfg.InsecureRegistries)...),
		imageinspector.NewSecretFetcher(secretClient, cache.NewInMemoryCache[*corev1.Secret](), imageinspector.WithSecretCacheTTL(cfg.TestkubeImageCredentialsCacheTTL)),
		inspectorStorages...,
	)
}

func CronJobsEnabled(cfg *config.Config) bool {
	enableCronJobs := cfg.EnableCronJobs
	if enableCronJobs == "" {
		var err error
		enableCronJobs, err = parser.LoadConfigFromFile(
			cfg.TestkubeConfigDir,
			"enable-cron-jobs",
			"enable cron jobs",
		)
		if err != nil {
			return false
		}
	}

	if enableCronJobs == "" {
		return false
	}

	result, err := strconv.ParseBool(enableCronJobs)
	if err != nil {
		return false
	}

	return result
}
