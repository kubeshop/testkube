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
	DeleteTests(selector string) error
	ListTests(selector string) (tests testkube.Tests, err error)
	ListTestWithExecutions(selector string) (tests testkube.TestWithExecutions, err error)
	ExecuteTest(id, executionName string, options ExecuteTestOptions) (executions testkube.Execution, err error)
	ExecuteTests(selector string, concurrencyLevel int, options ExecuteTestOptions) (executions []testkube.Execution, err error)
	Logs(id string) (logs chan output.Output, err error)

	CreateExecutor(options CreateExecutorOptions) (executor testkube.ExecutorDetails, err error)
	GetExecutor(name string) (executor testkube.ExecutorDetails, err error)
	ListExecutors(selector string) (executors testkube.ExecutorsDetails, err error)
	DeleteExecutor(name string) (err error)
	DeleteExecutors(selector string) (err error)

	CreateWebhook(options CreateWebhookOptions) (webhook testkube.Webhook, err error)
	GetWebhook(name string) (webhook testkube.Webhook, err error)
	ListWebhooks(selector string) (executors testkube.Webhooks, err error)
	DeleteWebhook(name string) (err error)
	DeleteWebhooks(selector string) (err error)

	GetExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error)
	DownloadFile(executionID, fileName, destination string) (artifact string, err error)

	CreateTestSuite(options UpsertTestSuiteOptions) (testSuite testkube.TestSuite, err error)
	UpdateTestSuite(options UpsertTestSuiteOptions) (testSuite testkube.TestSuite, err error)
	GetTestSuite(id string) (testSuite testkube.TestSuite, err error)
	GetTestSuiteWithExecution(id string) (testSuite testkube.TestSuiteWithExecution, err error)
	ListTestSuites(selector string) (testSuites testkube.TestSuites, err error)
	ListTestSuiteWithExecutions(selector string) (testSuitesWithExecutions testkube.TestSuiteWithExecutions, err error)
	DeleteTestSuite(name string) error
	DeleteTestSuites(selector string) error
	ExecuteTestSuite(id, executionName string, options ExecuteTestSuiteOptions) (executions testkube.TestSuiteExecution, err error)
	ExecuteTestSuites(selector string, concurrencyLevel int, options ExecuteTestSuiteOptions) (executions []testkube.TestSuiteExecution, err error)

	GetTestSuiteExecution(executionID string) (execution testkube.TestSuiteExecution, err error)
	ListTestSuiteExecutions(test string, limit int, selector string) (executions testkube.TestSuiteExecutionsResult, err error)
	WatchTestSuiteExecution(executionID string) (execution chan testkube.TestSuiteExecution, err error)

	GetServerInfo() (info testkube.ServerInfo, err error)
}

// TODO consider replacing below types by testkube.*

// UpsertTestSuiteOptions - mapping to OpenAPI schema for creating/changing testsuite
type UpsertTestSuiteOptions testkube.TestSuiteUpsertRequest

// UpsertTestOptions - is mapping for now to OpenAPI schema for creating/changing test
// if needed can beextended to custom struct
type UpsertTestOptions testkube.TestUpsertRequest

// CreateExecutorOptions - is mapping for now to OpenAPI schema for creating executor request
type CreateExecutorOptions testkube.ExecutorCreateRequest

// CreateWebhookOptions - is mapping for now to OpenAPI schema for creating/changing webhook
type CreateWebhookOptions testkube.WebhookCreateRequest

// TODO consider replacing it with testkube.ExecutionRequest - looks almost the samea and redundant
// ExecuteTestOptions contains test run options
type ExecuteTestOptions struct {
	ExecutionVariables            map[string]testkube.Variable
	ExecutionVariablesFileContent string
	Args                          []string
	Envs                          map[string]string
	SecretEnvs                    map[string]string
	HTTPProxy                     string
	HTTPSProxy                    string
}

// ExecuteTestSuiteOptions contains test suite run options
type ExecuteTestSuiteOptions struct {
	ExecutionVariables map[string]testkube.Variable
	HTTPProxy          string
	HTTPSProxy         string
}
