package services

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"

	kubeclient "github.com/kubeshop/testkube-operator/pkg/client"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	deprecatedapiv1 "github.com/kubeshop/testkube/internal/app/api/deprecatedv1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	parser "github.com/kubeshop/testkube/internal/template"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudartifacts "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	kubeexecutor "github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/containerexecutor"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/reconciler"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/tcl/checktcl"
	"github.com/kubeshop/testkube/pkg/tcl/schedulertcl"
)

type DeprecatedSystem struct {
	Clients      commons.DeprecatedClients
	Repositories commons.DeprecatedRepositories
	Scheduler    *scheduler.Scheduler
	Reconciler   *reconciler.Client
	JobExecutor  client.Executor
	API          *deprecatedapiv1.DeprecatedTestkubeAPI
	StreamLogs   func(ctx context.Context, executionID string) (chan output.Output, error)
}

func CreateDeprecatedSystem(
	ctx context.Context,
	mode string,
	cfg *config.Config,
	features featureflags.FeatureFlags,
	metrics metrics.Metrics,
	configMapConfig configRepo.Repository,
	secretConfig testkube.SecretConfig,
	grpcConn *grpc.ClientConn,
	natsConn *nats.EncodedConn,
	eventsEmitter *event.Emitter,
	eventBus *bus.NATSBus,
	inspector imageinspector.Inspector,
	subscriptionChecker checktcl.SubscriptionChecker,
	proContext *config.ProContext,
) *DeprecatedSystem {
	kubeClient, err := kubeclient.GetClient()
	commons.ExitOnError("Getting kubernetes client", err)
	clientset, err := k8sclient.ConnectToK8s()
	commons.ExitOnError("Creating k8s clientset", err)

	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)

	secretClient := secret.NewClientFor(clientset, cfg.TestkubeNamespace)
	configMapClient := configmap.NewClientFor(clientset, cfg.TestkubeNamespace)

	deprecatedClients := commons.CreateDeprecatedClients(kubeClient, cfg.TestkubeNamespace)
	deprecatedRepositories := commons.CreateDeprecatedRepositoriesForCloud(grpcClient, grpcConn, cfg.TestkubeProAPIKey)

	defaultExecutors, images, err := commons.ReadDefaultExecutors(cfg)
	commons.ExitOnError("Parsing default executors", err)
	if !cfg.TestkubeReadonlyExecutors {
		err := kubeexecutor.SyncDefaultExecutors(deprecatedClients.Executors(), cfg.TestkubeNamespace, defaultExecutors)
		commons.ExitOnError("Sync default executors", err)
	}
	jobTemplates, err := parser.ParseJobTemplates(cfg)
	commons.ExitOnError("Creating job templates", err)
	containerTemplates, err := parser.ParseContainerTemplates(cfg)
	commons.ExitOnError("Creating container job templates", err)

	serviceAccountNames := map[string]string{
		cfg.TestkubeNamespace: cfg.JobServiceAccountName,
	}
	// Pro edition only (tcl protected code)
	if cfg.TestkubeExecutionNamespaces != "" {
		err = subscriptionChecker.IsActiveOrgPlanEnterpriseForFeature("execution namespace")
		commons.ExitOnError("Subscription checking", err)
		serviceAccountNames = schedulertcl.GetServiceAccountNamesFromConfig(serviceAccountNames, cfg.TestkubeExecutionNamespaces)
	}

	clusterId, _ := configMapConfig.GetUniqueClusterId(ctx)

	var logGrpcClient logsclient.StreamGetter
	var logsStream logsclient.Stream
	if features.LogsV2 {
		logGrpcClient = commons.MustGetLogsV2Client(cfg)
		logsStream, err = logsclient.NewNatsLogStream(natsConn.Conn)
		commons.ExitOnError("Creating logs streaming client", err)
	}

	executor, err := client.NewJobExecutor(
		deprecatedRepositories,
		deprecatedClients,
		images,
		jobTemplates,
		serviceAccountNames,
		metrics,
		eventsEmitter,
		configMapConfig,
		clientset,
		cfg.TestkubeRegistry,
		cfg.TestkubePodStartTimeout,
		clusterId,
		cfg.TestkubeDashboardURI,
		fmt.Sprintf("http://%s:%d", cfg.APIServerFullname, cfg.APIServerPort),
		cfg.NatsURI,
		cfg.Debug,
		logsStream,
		features,
		cfg.TestkubeDefaultStorageClassName,
		cfg.WhitelistedContainers,
	)
	commons.ExitOnError("Creating executor client", err)

	containerExecutor, err := containerexecutor.NewContainerExecutor(
		deprecatedRepositories,
		deprecatedClients,
		images,
		containerTemplates,
		inspector,
		serviceAccountNames,
		metrics,
		eventsEmitter,
		configMapConfig,
		cfg.TestkubeRegistry,
		cfg.TestkubePodStartTimeout,
		clusterId,
		cfg.TestkubeDashboardURI,
		fmt.Sprintf("http://%s:%d", cfg.APIServerFullname, cfg.APIServerPort),
		cfg.NatsURI,
		cfg.Debug,
		logsStream,
		features,
		cfg.TestkubeDefaultStorageClassName,
		cfg.WhitelistedContainers,
		cfg.TestkubeImageCredentialsCacheTTL,
	)
	commons.ExitOnError("Creating container executor", err)

	sched := scheduler.NewScheduler(
		metrics,
		executor,
		containerExecutor,
		deprecatedRepositories,
		deprecatedClients,
		secretClient,
		eventsEmitter,
		log.DefaultLogger,
		configMapConfig,
		configMapClient,
		eventBus,
		cfg.TestkubeDashboardURI,
		features,
		logsStream,
		cfg.TestkubeNamespace,
		cfg.TestkubeProTLSSecret,
		cfg.TestkubeProRunnerCustomCASecret,
		subscriptionChecker,
	)

	storageParams := deprecatedapiv1.StorageParams{
		SSL:             cfg.StorageSSL,
		SkipVerify:      cfg.StorageSkipVerify,
		CertFile:        cfg.StorageCertFile,
		KeyFile:         cfg.StorageKeyFile,
		CAFile:          cfg.StorageCAFile,
		Endpoint:        cfg.StorageEndpoint,
		AccessKeyId:     cfg.StorageAccessKeyID,
		SecretAccessKey: cfg.StorageSecretAccessKey,
		Region:          cfg.StorageRegion,
		Token:           cfg.StorageToken,
		Bucket:          cfg.StorageBucket,
	}
	// Use direct MinIO artifact storage for deprecated API for backwards compatibility
	var deprecatedArtifactStorage storage.ArtifactsStorage
	if mode == common.ModeAgent {
		deprecatedArtifactStorage = cloudartifacts.NewCloudArtifactsStorage(grpcClient, grpcConn, cfg.TestkubeProAPIKey)
	} else {
		deprecatedArtifactStorage = minio.NewMinIOArtifactClient(commons.MustGetMinioClient(cfg))
	}
	deprecatedApi := deprecatedapiv1.NewDeprecatedTestkubeAPI(
		deprecatedRepositories,
		deprecatedClients,
		cfg.TestkubeNamespace,
		secretClient,
		eventsEmitter,
		executor,
		containerExecutor,
		metrics,
		sched,
		cfg.GraphqlPort,
		deprecatedArtifactStorage,
		mode,
		eventBus,
		secretConfig,
		features,
		logsStream,
		logGrpcClient,
		proContext,
		storageParams,
	)

	var reconcilerClient *reconciler.Client
	if !cfg.DisableReconciler {
		reconcilerClient = reconciler.NewClient(clientset, deprecatedRepositories, deprecatedClients, log.DefaultLogger)
	} else {
		log.DefaultLogger.Info("reconciler is disabled")
	}

	return &DeprecatedSystem{
		Clients:      deprecatedClients,
		Repositories: deprecatedRepositories,
		Scheduler:    sched,
		Reconciler:   reconcilerClient,
		API:          &deprecatedApi,
		StreamLogs:   deprecatedApi.GetLogsStream,
	}
}
