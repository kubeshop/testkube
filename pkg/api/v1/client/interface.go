package client

import (
	"io"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
	Do(req *http.Request) (resp *http.Response, err error)
}

type Client interface {
	GetExecution(executionID string) (execution testkube.Execution, err error)
	ListExecutions(id string, limit int, tags []string) (executions testkube.ExecutionsResult, err error)
	AbortExecution(test string, id string) error

	GetTest(id, namespace string) (test testkube.Test, err error)
	CreateTest(options UpsertTestOptions) (test testkube.Test, err error)
	UpdateTest(options UpsertTestOptions) (test testkube.Test, err error)
	DeleteTest(name string, namespace string) error
	DeleteTests(namespace string) error
	ListTests(namespace string, tags []string) (tests testkube.Tests, err error)
	ExecuteTest(id, namespace, executionName string, executionParams map[string]string, executionParamsFileContent string) (execution testkube.Execution, err error)
	Logs(id string) (logs chan output.Output, err error)

	CreateExecutor(options CreateExecutorOptions) (executor testkube.ExecutorDetails, err error)
	GetExecutor(name string) (executor testkube.ExecutorDetails, err error)
	ListExecutors() (executors testkube.ExecutorsDetails, err error)
	DeleteExecutor(name string) (err error)

	GetExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error)
	DownloadFile(executionID, fileName, destination string) (artifact string, err error)

	CreateTestSuite(options UpsertTestSuiteOptions) (testSuite testkube.TestSuite, err error)
	UpdateTestSuite(options UpsertTestSuiteOptions) (testSuite testkube.TestSuite, err error)
	GetTestSuite(id string, namespace string) (testSuite testkube.TestSuite, err error)
	ListTestSuites(namespace string, tags []string) (testSuites testkube.TestSuites, err error)
	DeleteTestSuite(name string, namespace string) error
	DeleteTestSuites(namespace string) error
	ExecuteTestSuite(id, namespace, executionName string, executionParams map[string]string) (execution testkube.TestSuiteExecution, err error)

	GetTestSuiteExecution(executionID string) (execution testkube.TestSuiteExecution, err error)
	ListTestSuiteExecutions(test string, limit int, tags []string) (executions testkube.TestSuiteExecutionsResult, err error)
	WatchTestSuiteExecution(executionID string) (execution chan testkube.TestSuiteExecution, err error)

	GetServerInfo() (info testkube.ServerInfo, err error)
}

type UpsertTestSuiteOptions testkube.TestSuiteUpsertRequest

// UpsertTestOptions - is mapping for now to OpenAPI schema for creating request
// if needed can beextended to custom struct
type UpsertTestOptions testkube.TestUpsertRequest

// CreateExectorOptions - is mapping for now to OpenAPI schema for creating request
type CreateExecutorOptions testkube.ExecutorCreateRequest
