package commons

import (
	executorsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/executors/v1"
	templatesclientv1 "github.com/kubeshop/testkube/pkg/operator/client/templates/v1"
	testexecutionsclientv1 "github.com/kubeshop/testkube/pkg/operator/client/testexecutions/v1"
	testsclientv3 "github.com/kubeshop/testkube/pkg/operator/client/tests/v3"
	testsourcesclientv1 "github.com/kubeshop/testkube/pkg/operator/client/testsources/v1"
	testsuiteexecutionsv1 "github.com/kubeshop/testkube/pkg/operator/client/testsuiteexecutions/v1"
	testsuitesclientv3 "github.com/kubeshop/testkube/pkg/operator/client/testsuites/v3"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudresult "github.com/kubeshop/testkube/pkg/cloud/data/result"
	cloudtestresult "github.com/kubeshop/testkube/pkg/cloud/data/testresult"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/repository"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
)

//go:generate go tool mockgen -destination=./mock_deprecatedclients.go -package=commons "github.com/kubeshop/testkube/cmd/api-server/commons" DeprecatedClients
type DeprecatedClients interface {
	Executors() executorsclientv1.Interface
	Tests() testsclientv3.Interface
	TestSuites() testsuitesclientv3.Interface
	TestSources() testsourcesclientv1.Interface
	TestExecutions() testexecutionsclientv1.Interface
	TestSuiteExecutions() testsuiteexecutionsv1.Interface
	Templates() templatesclientv1.Interface
}

//go:generate go tool mockgen -destination=./mock_deprecatedrepositories.go -package=commons "github.com/kubeshop/testkube/cmd/api-server/commons" DeprecatedRepositories
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

func CreateDeprecatedRepositoriesForMongo(repoManager repository.DatabaseRepository) DeprecatedRepositories {
	return &deprecatedRepositories{
		testResults:      repoManager.Result(),
		testSuiteResults: repoManager.TestResult(),
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
