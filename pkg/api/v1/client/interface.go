package client

import (
	"io"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

// HTTPClient abstracts http client methods
type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
	Do(req *http.Request) (resp *http.Response, err error)
}

// Client is the Testkube API client abstraction
type Client interface {
	GetExecution(executionID string) (execution testkube.Execution, err error)
	ListExecutions(id string, limit int, selector string) (executions testkube.ExecutionsResult, err error)
	AbortExecution(test string, id string) error

	GetTest(id string) (test testkube.Test, err error)
	GetTestWithExecution(id string) (test testkube.TestWithExecution, err error)
	CreateTest(options UpsertTestOptions) (test testkube.Test, err error)
	UpdateTest(options UpsertTestOptions) (test testkube.Test, err error)
	DeleteTest(name string) error
	DeleteTests() error
	ListTests(selector string) (tests testkube.Tests, err error)
	ExecuteTest(id, executionName string, executionParams map[string]string, executionParamsFileContent string, args []string) (execution testkube.Execution, err error)
	Logs(id string) (logs chan output.Output, err error)

	CreateExecutor(options CreateExecutorOptions) (executor testkube.ExecutorDetails, err error)
	GetExecutor(name string) (executor testkube.ExecutorDetails, err error)
	ListExecutors() (executors testkube.ExecutorsDetails, err error)
	DeleteExecutor(name string) (err error)

	CreateWebhook(options CreateWebhookOptions) (webhook testkube.Webhook, err error)
	GetWebhook(name string) (webhook testkube.Webhook, err error)
	ListWebhooks() (executors testkube.Webhooks, err error)
	DeleteWebhook(name string) (err error)

	GetExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error)
	DownloadFile(executionID, fileName, destination string) (artifact string, err error)

	CreateTestSuite(options UpsertTestSuiteOptions) (testSuite testkube.TestSuite, err error)
	UpdateTestSuite(options UpsertTestSuiteOptions) (testSuite testkube.TestSuite, err error)
	GetTestSuite(id string) (testSuite testkube.TestSuite, err error)
	GetTestSuiteWithExecution(id string) (testSuite testkube.TestSuiteWithExecution, err error)
	ListTestSuites(selector string) (testSuites testkube.TestSuites, err error)
	DeleteTestSuite(name string) error
	DeleteTestSuites() error
	ExecuteTestSuite(id, executionName string, executionParams map[string]string) (execution testkube.TestSuiteExecution, err error)

	GetTestSuiteExecution(executionID string) (execution testkube.TestSuiteExecution, err error)
	ListTestSuiteExecutions(test string, limit int, selector string) (executions testkube.TestSuiteExecutionsResult, err error)
	WatchTestSuiteExecution(executionID string) (execution chan testkube.TestSuiteExecution, err error)

	GetServerInfo() (info testkube.ServerInfo, err error)
}

// UpsertTestSuiteOptions - mapping to OpenAPI schema for creating/changing testsuite
type UpsertTestSuiteOptions testkube.TestSuiteUpsertRequest

// UpsertTestOptions - is mapping for now to OpenAPI schema for creating/changing test
// if needed can beextended to custom struct
type UpsertTestOptions testkube.TestUpsertRequest

// CreateExectorOptions - is mapping for now to OpenAPI schema for creating executor request
type CreateExecutorOptions testkube.ExecutorCreateRequest

// CreateExectorOptions - is mapping for now to OpenAPI schema for creating/changing webhook
type CreateWebhookOptions testkube.WebhookCreateRequest
