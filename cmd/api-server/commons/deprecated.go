package commons

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"sigs.k8s.io/controller-runtime/pkg/client"

	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	templatesclientv1 "github.com/kubeshop/testkube-operator/pkg/client/templates/v1"
	testexecutionsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testexecutions/v1"
	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	testsourcesclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testsources/v1"
	testsuiteexecutionsv1 "github.com/kubeshop/testkube-operator/pkg/client/testsuiteexecutions/v1"
	testsuitesclientv3 "github.com/kubeshop/testkube-operator/pkg/client/testsuites/v3"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudresult "github.com/kubeshop/testkube/pkg/cloud/data/result"
	cloudtestresult "github.com/kubeshop/testkube/pkg/cloud/data/testresult"
	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/log"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/repository/result"
	minioresult "github.com/kubeshop/testkube/pkg/repository/result/minio"
	mongoresult "github.com/kubeshop/testkube/pkg/repository/result/mongo"
	"github.com/kubeshop/testkube/pkg/repository/sequence"
	"github.com/kubeshop/testkube/pkg/repository/storage"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	mongotestresult "github.com/kubeshop/testkube/pkg/repository/testresult/mongo"
	domainstorage "github.com/kubeshop/testkube/pkg/storage"
)

//go:generate mockgen -destination=./mock_deprecatedclients.go -package=commons "github.com/kubeshop/testkube/cmd/api-server/commons" DeprecatedClients
type DeprecatedClients interface {
	Executors() executorsclientv1.Interface
	Tests() testsclientv3.Interface
	TestSuites() testsuitesclientv3.Interface
	TestSources() testsourcesclientv1.Interface
	TestExecutions() testexecutionsclientv1.Interface
	TestSuiteExecutions() testsuiteexecutionsv1.Interface
	Templates() templatesclientv1.Interface
}

type deprecatedClients struct {
	executors           executorsclientv1.Interface
	tests               testsclientv3.Interface
	testSuites          testsuitesclientv3.Interface
	testSources         testsourcesclientv1.Interface
	testExecutions      testexecutionsclientv1.Interface
	testSuiteExecutions testsuiteexecutionsv1.Interface
	templates           templatesclientv1.Interface
}

func (d *deprecatedClients) Executors() executorsclientv1.Interface {
	return d.executors
}

func (d *deprecatedClients) Tests() testsclientv3.Interface {
	return d.tests
}

func (d *deprecatedClients) TestSuites() testsuitesclientv3.Interface {
	return d.testSuites
}

func (d *deprecatedClients) TestSources() testsourcesclientv1.Interface {
	return d.testSources
}

func (d *deprecatedClients) TestExecutions() testexecutionsclientv1.Interface {
	return d.testExecutions
}

func (d *deprecatedClients) TestSuiteExecutions() testsuiteexecutionsv1.Interface {
	return d.testSuiteExecutions
}

func (d *deprecatedClients) Templates() templatesclientv1.Interface {
	return d.templates
}

// TODO: Move Templates() out of Deprecation, as it's used by Webhook Payload (?)
func CreateDeprecatedClients(kubeClient client.Client, namespace string) DeprecatedClients {
	return &deprecatedClients{
		executors:           executorsclientv1.NewClient(kubeClient, namespace),
		tests:               testsclientv3.NewClient(kubeClient, namespace),
		testSuites:          testsuitesclientv3.NewClient(kubeClient, namespace),
		testSources:         testsourcesclientv1.NewClient(kubeClient, namespace),
		testExecutions:      testexecutionsclientv1.NewClient(kubeClient, namespace),
		testSuiteExecutions: testsuiteexecutionsv1.NewClient(kubeClient, namespace),
		templates:           templatesclientv1.NewClient(kubeClient, namespace),
	}
}

//go:generate mockgen -destination=./mock_deprecatedrepositories.go -package=commons "github.com/kubeshop/testkube/cmd/api-server/commons" DeprecatedRepositories
type DeprecatedRepositories interface {
	TestResults() result.Repository
	TestSuiteResults() testresult.Repository
}

type deprecatedRepositories struct {
	testResults      result.Repository
	testSuiteResults testresult.Repository
}

func (d *deprecatedRepositories) TestResults() result.Repository {
	return d.testResults
}

func (d *deprecatedRepositories) TestSuiteResults() testresult.Repository {
	return d.testSuiteResults
}

func CreateDeprecatedRepositoriesForCloud(grpcClient cloud.TestKubeCloudAPIClient, apiKey string) DeprecatedRepositories {
	return &deprecatedRepositories{
		testResults:      cloudresult.NewCloudResultRepository(grpcClient, apiKey),
		testSuiteResults: cloudtestresult.NewCloudRepository(grpcClient, apiKey),
	}
}

func CreateDeprecatedRepositoriesForMongo(ctx context.Context, cfg *config.Config, db *mongo.Database, logGrpcClient logsclient.StreamGetter, storageClient domainstorage.Client, features featureflags.FeatureFlags) DeprecatedRepositories {
	isDocDb := cfg.APIMongoDBType == storage.TypeDocDB
	sequenceRepository := sequence.NewMongoRepository(db)
	mongoResultsRepository := mongoresult.NewMongoRepository(db, cfg.APIMongoAllowDiskUse, isDocDb, mongoresult.WithFeatureFlags(features),
		mongoresult.WithLogsClient(logGrpcClient), mongoresult.WithMongoRepositorySequence(sequenceRepository))
	resultsRepository := mongoResultsRepository
	testResultsRepository := mongotestresult.NewMongoRepository(db, cfg.APIMongoAllowDiskUse, isDocDb,
		mongotestresult.WithMongoRepositorySequence(sequenceRepository))

	// Init logs storage
	if cfg.LogsStorage == "minio" {
		if cfg.LogsBucket == "" {
			log.DefaultLogger.Error("LOGS_BUCKET env var is not set")
		} else if ok, err := storageClient.IsConnectionPossible(ctx); ok && (err == nil) {
			log.DefaultLogger.Info("setting minio as logs storage")
			mongoResultsRepository.OutputRepository = minioresult.NewMinioOutputRepository(storageClient, mongoResultsRepository.ResultsColl, cfg.LogsBucket)
		} else {
			log.DefaultLogger.Infow("minio is not available, using default logs storage", "error", err)
		}
	}

	return &deprecatedRepositories{
		testResults:      resultsRepository,
		testSuiteResults: testResultsRepository,
	}
}

func MustGetLogsV2Client(cfg *config.Config) logsclient.StreamGetter {
	creds, err := logsclient.GetGrpcTransportCredentials(logsclient.GrpcConnectionConfig{
		Secure:     cfg.LogServerSecure,
		SkipVerify: cfg.LogServerSkipVerify,
		CertFile:   cfg.LogServerCertFile,
		KeyFile:    cfg.LogServerKeyFile,
		CAFile:     cfg.LogServerCAFile,
	})
	ExitOnError("Getting log server TLS credentials", err)
	return logsclient.NewGrpcClient(cfg.LogServerGrpcAddress, creds)
}
