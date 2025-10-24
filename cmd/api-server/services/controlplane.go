package services

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/controlplane"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	database "github.com/kubeshop/testkube/pkg/database/postgres"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	kubeclient "github.com/kubeshop/testkube/pkg/operator/client"
	"github.com/kubeshop/testkube/pkg/repository"
	minioresult "github.com/kubeshop/testkube/pkg/repository/result/minio"
	"github.com/kubeshop/testkube/pkg/repository/storage"
	miniorepo "github.com/kubeshop/testkube/pkg/repository/testworkflow/minio"
	"github.com/kubeshop/testkube/pkg/secret"
	domainstorage "github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

func CreateControlPlane(ctx context.Context, cfg *config.Config, features featureflags.FeatureFlags, eventsEmitter *event.Emitter) *controlplane.Server {
	// Connect to the cluster
	kubeConfig, err := k8sclient.GetK8sClientConfig()
	commons.ExitOnError("Getting kubernetes config", err)
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	commons.ExitOnError("Creating k8s clientset", err)
	kubeClient, err := kubeclient.GetClient()
	commons.ExitOnError("Getting kubernetes client", err)

	// Connect to storages
	secretClient := secret.NewClientFor(clientset, cfg.TestkubeNamespace)
	storageClient := commons.MustGetMinioClient(cfg)

	var logGrpcClient logsclient.StreamGetter
	if !cfg.DisableDeprecatedTests && features.LogsV2 {
		logGrpcClient = commons.MustGetLogsV2Client(cfg)
		commons.ExitOnError("Creating logs streaming client", err)
	}

	var factory repository.RepositoryFactory
	if cfg.APIMongoDSN != "" {
		mongoDb := commons.MustGetMongoDatabase(ctx, cfg, secretClient, !cfg.DisableMongoMigrations)
		factory, err = CreateMongoFactory(ctx, cfg, mongoDb, logGrpcClient, storageClient)
	}
	if cfg.APIPostgresDSN != "" {
		postgresDb := commons.MustGetPostgresDatabase(ctx, cfg, !cfg.DisablePostgresMigrations)
		factory, err = CreatePostgresFactory(postgresDb)
	}
	commons.ExitOnError("Creating factory for database", err)

	testWorkflowsClient, err := testworkflowclient.NewKubernetesTestWorkflowClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)
	commons.ExitOnError("Creating test workflow client", err)
	testWorkflowTemplatesClient, err := testworkflowtemplateclient.NewKubernetesTestWorkflowTemplateClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)
	commons.ExitOnError("Creating test workflow templates client", err)

	// Build repositories
	repoManager := repository.NewRepositoryManager(factory)
	testWorkflowResultsRepository := repoManager.TestWorkflow()
	testWorkflowOutputRepository := miniorepo.NewMinioOutputRepository(storageClient, testWorkflowResultsRepository, cfg.LogsBucket)
	deprecatedRepositories := commons.CreateDeprecatedRepositoriesForMongo(repoManager)
	artifactStorage := minio.NewMinIOArtifactClient(storageClient)
	commands := controlplane.CreateCommands(cfg.DisableDeprecatedTests, cfg.StorageBucket, deprecatedRepositories, storageClient, testWorkflowOutputRepository, testWorkflowResultsRepository, artifactStorage)

	enqueuer := scheduling.NewEnqueuer(log.DefaultLogger, testWorkflowsClient, testWorkflowTemplatesClient, testWorkflowResultsRepository, eventsEmitter)
	scheduler := factory.NewScheduler()
	executionController := factory.NewExecutionController()
	executionQuerier := factory.NewExecutionQuerier()

	// Ensure the buckets exist
	if cfg.StorageBucket != "" {
		exists, err := storageClient.BucketExists(ctx, cfg.StorageBucket)
		if err != nil {
			log.DefaultLogger.Errorw("Failed to check if the storage bucket exists", "error", err)
		} else if !exists {
			err = storageClient.CreateBucket(ctx, cfg.StorageBucket)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				log.DefaultLogger.Errorw("Creating storage bucket", "error", err)
			}
		}
	}
	if cfg.LogsBucket != "" {
		exists, err := storageClient.BucketExists(ctx, cfg.LogsBucket)
		if err != nil {
			log.DefaultLogger.Errorw("Failed to check if the storage bucket exists", "error", err)
		} else if !exists {
			err = storageClient.CreateBucket(ctx, cfg.LogsBucket)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				log.DefaultLogger.Errorw("Creating logs bucket", "error", err)
			}
		}
	}

	return controlplane.New(controlplane.Config{
		Port:                             cfg.GRPCServerPort,
		Logger:                           log.DefaultLogger,
		Verbose:                          false,
		StorageBucket:                    cfg.StorageBucket,
		FeatureTestWorkflowsCloudStorage: cfg.FeatureCloudStorage,
	}, enqueuer, scheduler, executionController, executionQuerier, eventsEmitter, storageClient, testWorkflowsClient, testWorkflowTemplatesClient,
		testWorkflowResultsRepository, testWorkflowOutputRepository, repoManager, commands...)
}

func CreateMongoFactory(ctx context.Context, cfg *config.Config, db *mongo.Database,
	logGrpcClient logsclient.StreamGetter, storageClient domainstorage.Client) (repository.RepositoryFactory, error) {
	var outputRepository *minioresult.MinioRepository
	// Init logs storage
	if cfg.LogsStorage == "minio" {
		if cfg.LogsBucket == "" {
			log.DefaultLogger.Error("LOGS_BUCKET env var is not set")
		} else if ok, err := storageClient.IsConnectionPossible(ctx); ok && (err == nil) {
			log.DefaultLogger.Info("setting minio as logs storage")
			outputRepository = minioresult.NewMinioOutputRepository(storageClient, cfg.LogsBucket)
		} else {
			log.DefaultLogger.Infow("minio is not available, using default logs storage", "error", err)
		}
	}

	factory, err := repository.NewFactoryBuilder().WithMongoDB(repository.MongoDBFactoryConfig{
		Database:         db,
		AllowDiskUse:     cfg.APIMongoAllowDiskUse,
		IsDocDb:          cfg.APIMongoDBType == storage.TypeDocDB,
		LogGrpcClient:    logGrpcClient,
		OutputRepository: outputRepository,
	}).Build()
	if err != nil {
		return nil, err
	}

	return factory, nil
}

func CreatePostgresFactory(db *pgxpool.Pool) (repository.RepositoryFactory, error) {
	schedulerDb, err := database.NewForScheduler(db)
	if err != nil {
		return nil, err
	}

	factory, err := repository.NewFactoryBuilder().WithPostgreSQL(repository.PostgreSQLFactoryConfig{
		Database:    db,
		SchedulerDb: schedulerDb,
	}).Build()
	if err != nil {
		return nil, err
	}

	return factory, nil
}
