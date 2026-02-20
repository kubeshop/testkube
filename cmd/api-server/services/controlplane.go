package services

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/controlplane"
	"github.com/kubeshop/testkube/pkg/controlplane/scheduling"
	database "github.com/kubeshop/testkube/pkg/database/postgres"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	kubeclient "github.com/kubeshop/testkube/pkg/operator/client"
	"github.com/kubeshop/testkube/pkg/repository"
	"github.com/kubeshop/testkube/pkg/repository/storage"
	miniorepo "github.com/kubeshop/testkube/pkg/repository/testworkflow/minio"
	"github.com/kubeshop/testkube/pkg/secret"
	domainstorage "github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

func CreateControlPlane(ctx context.Context, cfg *config.Config, eventsEmitter *event.Emitter, envID string) *controlplane.Server {
	// Connect to the cluster
	kubeConfig, err := k8sclient.GetK8sClientConfig()
	commons.ExitOnError("Getting kubernetes config", err)
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	commons.ExitOnError("Creating k8s clientset", err)
	kubeClient, err := kubeclient.GetClient()
	commons.ExitOnError("Getting kubernetes client", err)

	// Connect to storages
	secretClient := secret.NewClientFor(clientset, cfg.TestkubeNamespace)

	var factory repository.RepositoryFactory
	if cfg.APIMongoDSN != "" {
		mongoDb := commons.MustGetMongoDatabase(ctx, cfg, secretClient, !cfg.DisableMongoMigrations)
		factory, err = CreateMongoFactory(ctx, cfg, mongoDb)
	}
	if cfg.APIPostgresDSN != "" {
		postgresDb := commons.MustGetPostgresDatabase(ctx, cfg, !cfg.DisablePostgresMigrations)
		factory, err = CreatePostgresFactory(postgresDb)
	}
	commons.ExitOnError("Creating factory for database", err)

	testWorkflowsClient, err := testworkflowclient.NewKubernetesTestWorkflowClient(kubeClient, kubeConfig, cfg.TestkubeNamespace)
	commons.ExitOnError("Creating test workflow client", err)
	testWorkflowTemplatesClient, err := testworkflowtemplateclient.NewKubernetesTestWorkflowTemplateClient(kubeClient, kubeConfig, cfg.TestkubeNamespace,
		cfg.DisableOfficialTemplates, cfg.GlobalWorkflowTemplateInline)
	commons.ExitOnError("Creating test workflow templates client", err)

	// Build repositories
	repoManager := repository.NewRepositoryManager(factory)
	testWorkflowResultsRepository := repoManager.TestWorkflow()
	storageClient := commons.MustGetMinioClient(cfg)
	testWorkflowOutputRepository := miniorepo.NewMinioOutputRepository(storageClient, testWorkflowResultsRepository, cfg.LogsBucket)
	artifactStorage := minio.NewMinIOArtifactClient(storageClient)
	commands := controlplane.CreateCommands(cfg.StorageBucket, storageClient, testWorkflowOutputRepository, testWorkflowResultsRepository, artifactStorage)

	enqueuer := scheduling.NewEnqueuer(log.DefaultLogger, testWorkflowsClient, testWorkflowTemplatesClient, testWorkflowResultsRepository, eventsEmitter,
		cfg.GlobalWorkflowTemplateName, cfg.GlobalWorkflowTemplateInline != "")
	scheduler := factory.NewScheduler()
	executionController := factory.NewExecutionController()
	executionQuerier := factory.NewExecutionQuerier()

	// Ensure the buckets exist (retry in background until they do).
	go ensureBucketsWithRetry(ctx, storageClient, []bucketSpec{
		{name: cfg.StorageBucket, label: "storage"},
		{name: cfg.LogsBucket, label: "logs"},
	})

	return controlplane.New(controlplane.Config{
		Port:                             cfg.GRPCServerPort,
		Logger:                           log.DefaultLogger,
		Verbose:                          false,
		StorageBucket:                    cfg.StorageBucket,
		FeatureTestWorkflowsCloudStorage: cfg.FeatureCloudStorage,
	}, enqueuer, scheduler, executionController, executionQuerier, eventsEmitter, storageClient, testWorkflowsClient, testWorkflowTemplatesClient,
		testWorkflowResultsRepository, testWorkflowOutputRepository, repoManager, envID, commands...)
}

type bucketSpec struct {
	name  string
	label string
}

func ensureBucketsWithRetry(ctx context.Context, storageClient domainstorage.Client, buckets []bucketSpec) {
	var active []bucketSpec
	for _, bucket := range buckets {
		if bucket.name != "" {
			active = append(active, bucket)
		}
	}
	if len(active) == 0 {
		return
	}

	delay := 1 * time.Second
	maxDelay := 30 * time.Second

	for {
		remaining := 0
		for _, bucket := range active {
			if !ensureBucket(ctx, storageClient, bucket) {
				remaining++
			}
		}

		if remaining == 0 {
			return
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
}

func ensureBucket(ctx context.Context, storageClient domainstorage.Client, bucket bucketSpec) bool {
	exists, err := storageClient.BucketExists(ctx, bucket.name)
	if err != nil {
		log.DefaultLogger.Warnw("Failed to check if the bucket exists; will retry", "bucket", bucket.name, "label", bucket.label, "error", err)
		return false
	}
	if exists {
		return true
	}
	if err := storageClient.CreateBucket(ctx, bucket.name); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return true
		}
		log.DefaultLogger.Warnw("Creating bucket failed; will retry", "bucket", bucket.name, "label", bucket.label, "error", err)
		return false
	}
	log.DefaultLogger.Infow("Created bucket", "bucket", bucket.name, "label", bucket.label)
	return true
}

func CreateMongoFactory(_ context.Context, cfg *config.Config, db *mongo.Database) (repository.RepositoryFactory, error) {

	factory, err := repository.NewFactoryBuilder().WithMongoDB(repository.MongoDBFactoryConfig{
		Database:     db,
		AllowDiskUse: cfg.APIMongoAllowDiskUse,
		IsDocDb:      cfg.APIMongoDBType == storage.TypeDocDB,
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
