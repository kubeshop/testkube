package commons

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

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

func CreateDeprecatedRepositoriesForCloud(grpcClient cloud.TestKubeCloudAPIClient, proContext *config.ProContext) DeprecatedRepositories {
	return &deprecatedRepositories{
		testResults:      cloudresult.NewCloudResultRepository(grpcClient, proContext),
		testSuiteResults: cloudtestresult.NewCloudRepository(grpcClient, proContext),
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
