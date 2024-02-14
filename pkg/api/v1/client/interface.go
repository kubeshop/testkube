package client

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/logs/events"
)

// Client is the Testkube API client abstraction
type Client interface {
	TestAPI
	ExecutionAPI
	TestSuiteAPI
	TestSuiteExecutionAPI
	ExecutorAPI
	WebhookAPI
	ServiceAPI
	ConfigAPI
	TestSourceAPI
	CopyFileAPI
	TemplateAPI
	TestWorkflowAPI
	TestWorkflowExecutionAPI
	TestWorkflowTemplateAPI
}

// TestAPI describes test api methods
type TestAPI interface {
	GetTest(id string) (test testkube.Test, err error)
	GetTestWithExecution(id string) (test testkube.TestWithExecution, err error)
	CreateTest(options UpsertTestOptions) (test testkube.Test, err error)
	UpdateTest(options UpdateTestOptions) (test testkube.Test, err error)
	DeleteTest(name string) error
	DeleteTests(selector string) error
	ListTests(selector string) (tests testkube.Tests, err error)
	ListTestWithExecutionSummaries(selector string) (tests testkube.TestWithExecutionSummaries, err error)
	ExecuteTest(id, executionName string, options ExecuteTestOptions) (executions testkube.Execution, err error)
	ExecuteTests(selector string, concurrencyLevel int, options ExecuteTestOptions) (executions []testkube.Execution, err error)
	Logs(id string) (logs chan output.Output, err error)
	LogsV2(id string) (logs chan events.Log, err error)
}

// ExecutionAPI describes execution api methods
type ExecutionAPI interface {
	GetExecution(executionID string) (execution testkube.Execution, err error)
	ListExecutions(id string, limit int, selector string) (executions testkube.ExecutionsResult, err error)
	AbortExecution(test string, id string) error
	AbortExecutions(test string) error
	GetExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error)
	DownloadFile(executionID, fileName, destination string) (artifact string, err error)
	DownloadArchive(executionID, destination string, masks []string) (archive string, err error)
}

// TestSuiteAPI describes test suite api methods
type TestSuiteAPI interface {
	CreateTestSuite(options UpsertTestSuiteOptions) (testSuite testkube.TestSuite, err error)
	UpdateTestSuite(options UpdateTestSuiteOptions) (testSuite testkube.TestSuite, err error)
	GetTestSuite(id string) (testSuite testkube.TestSuite, err error)
	GetTestSuiteWithExecution(id string) (testSuite testkube.TestSuiteWithExecution, err error)
	ListTestSuites(selector string) (testSuites testkube.TestSuites, err error)
	ListTestSuiteWithExecutionSummaries(selector string) (testSuitesWithExecutionSummaries testkube.TestSuiteWithExecutionSummaries, err error)
	DeleteTestSuite(name string) error
	DeleteTestSuites(selector string) error
	ExecuteTestSuite(id, executionName string, options ExecuteTestSuiteOptions) (executions testkube.TestSuiteExecution, err error)
	ExecuteTestSuites(selector string, concurrencyLevel int, options ExecuteTestSuiteOptions) (executions []testkube.TestSuiteExecution, err error)
}

// TestSuiteExecutionAPI describes test suite execution api methods
type TestSuiteExecutionAPI interface {
	GetTestSuiteExecution(executionID string) (execution testkube.TestSuiteExecution, err error)
	ListTestSuiteExecutions(testsuite string, limit int, selector string) (executions testkube.TestSuiteExecutionsResult, err error)
	WatchTestSuiteExecution(executionID string) (resp chan testkube.WatchTestSuiteExecutionResponse)
	AbortTestSuiteExecution(executionID string) error
	AbortTestSuiteExecutions(testSuiteName string) error
	GetTestSuiteExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error)
}

// ExecutorAPI describes executor api methods
type ExecutorAPI interface {
	CreateExecutor(options UpsertExecutorOptions) (executor testkube.ExecutorDetails, err error)
	UpdateExecutor(options UpdateExecutorOptions) (executor testkube.ExecutorDetails, err error)
	GetExecutor(name string) (executor testkube.ExecutorDetails, err error)
	ListExecutors(selector string) (executors testkube.ExecutorsDetails, err error)
	DeleteExecutor(name string) (err error)
	DeleteExecutors(selector string) (err error)
}

// WebhookAPI describes webhook api methods
type WebhookAPI interface {
	CreateWebhook(options CreateWebhookOptions) (webhook testkube.Webhook, err error)
	UpdateWebhook(options UpdateWebhookOptions) (webhook testkube.Webhook, err error)
	GetWebhook(name string) (webhook testkube.Webhook, err error)
	ListWebhooks(selector string) (webhooks testkube.Webhooks, err error)
	DeleteWebhook(name string) (err error)
	DeleteWebhooks(selector string) (err error)
}

// TemplateAPI describes template api methods
type TemplateAPI interface {
	CreateTemplate(options CreateTemplateOptions) (template testkube.Template, err error)
	UpdateTemplate(options UpdateTemplateOptions) (template testkube.Template, err error)
	GetTemplate(name string) (template testkube.Template, err error)
	ListTemplates(selector string) (templates testkube.Templates, err error)
	DeleteTemplate(name string) (err error)
	DeleteTemplates(selector string) (err error)
}

// ConfigAPI describes config api methods
type ConfigAPI interface {
	UpdateConfig(config testkube.Config) (outputConfig testkube.Config, err error)
	GetConfig() (config testkube.Config, err error)
}

// ServiceAPI describes service api methods
type ServiceAPI interface {
	GetServerInfo() (info testkube.ServerInfo, err error)
	GetDebugInfo() (info testkube.DebugInfo, err error)
}

// TestSourceAPI describes test source api methods
type TestSourceAPI interface {
	CreateTestSource(options UpsertTestSourceOptions) (testSource testkube.TestSource, err error)
	UpdateTestSource(options UpdateTestSourceOptions) (testSource testkube.TestSource, err error)
	GetTestSource(name string) (testSource testkube.TestSource, err error)
	ListTestSources(selector string) (testSources testkube.TestSources, err error)
	DeleteTestSource(name string) (err error)
	DeleteTestSources(selector string) (err error)
}

// TestWorkflowAPI describes test workflow api methods
type TestWorkflowAPI interface {
	GetTestWorkflow(id string) (testkube.TestWorkflow, error)
	GetTestWorkflowWithExecution(id string) (testkube.TestWorkflowWithExecution, error)
	ListTestWorkflows(selector string) (testkube.TestWorkflows, error)
	ListTestWorkflowWithExecutions(selector string) (testkube.TestWorkflowWithExecutions, error)
	DeleteTestWorkflows(selector string) error
	CreateTestWorkflow(workflow testkube.TestWorkflow) (testkube.TestWorkflow, error)
	UpdateTestWorkflow(workflow testkube.TestWorkflow) (testkube.TestWorkflow, error)
	DeleteTestWorkflow(name string) error
	ExecuteTestWorkflow(name string, request testkube.TestWorkflowExecutionRequest) (testkube.TestWorkflowExecution, error)
	GetTestWorkflowExecutionNotifications(id string) (chan testkube.TestWorkflowExecutionNotification, error)
}

// TestWorkflowExecutionAPI describes test workflow api methods
type TestWorkflowExecutionAPI interface {
	GetTestWorkflowExecution(executionID string) (execution testkube.TestWorkflowExecution, err error)
	ListTestWorkflowExecutions(id string, limit int, selector string) (executions testkube.TestWorkflowExecutionsResult, err error)
	AbortTestWorkflowExecution(workflow string, id string) error
	AbortTestWorkflowExecutions(workflow string) error
	GetTestWorkflowExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error)
	DownloadTestWorkflowArtifact(executionID, fileName, destination string) (artifact string, err error)
	DownloadTestWorkflowArtifactArchive(executionID, destination string, masks []string) (archive string, err error)
}

// TestWorkflowTemplateAPI describes test workflow api methods
type TestWorkflowTemplateAPI interface {
	GetTestWorkflowTemplate(id string) (testkube.TestWorkflowTemplate, error)
	ListTestWorkflowTemplates(selector string) (testkube.TestWorkflowTemplates, error)
	DeleteTestWorkflowTemplates(selector string) error
	CreateTestWorkflowTemplate(workflow testkube.TestWorkflowTemplate) (testkube.TestWorkflowTemplate, error)
	UpdateTestWorkflowTemplate(workflow testkube.TestWorkflowTemplate) (testkube.TestWorkflowTemplate, error)
	DeleteTestWorkflowTemplate(name string) error
}

// CopyFileAPI describes methods to handle files in the object storage
type CopyFileAPI interface {
	UploadFile(parentName string, parentType TestingType, filePath string, fileContent []byte, timeout time.Duration) error
}

// TODO consider replacing below types by testkube.*

// UpsertTestSuiteOptions - mapping to OpenAPI schema for creating testsuite
type UpsertTestSuiteOptions testkube.TestSuiteUpsertRequest

// UpdateTestSuiteOptions - mapping to OpenAPI schema for changing testsuite
type UpdateTestSuiteOptions testkube.TestSuiteUpdateRequest

// UpsertTestOptions - is mapping for now to OpenAPI schema for creating test
// if needed can be extended to custom struct
type UpsertTestOptions testkube.TestUpsertRequest

// UpdateTestOptions - is mapping for now to OpenAPI schema for changing test
// if needed can be extended to custom struct
type UpdateTestOptions testkube.TestUpdateRequest

// UpsertExecutorOptions - is mapping for now to OpenAPI schema for creating executor request
type UpsertExecutorOptions testkube.ExecutorUpsertRequest

// UpdateExecutorOptions - is mapping for now to OpenAPI schema for changing executor request
type UpdateExecutorOptions testkube.ExecutorUpdateRequest

// CreateWebhookOptions - is mapping for now to OpenAPI schema for creating/changing webhook
type CreateWebhookOptions testkube.WebhookCreateRequest

// UpdateWebhookOptions - is mapping for now to OpenAPI schema for changing webhook request
type UpdateWebhookOptions testkube.WebhookUpdateRequest

// UpsertTestSourceOptions - is mapping for now to OpenAPI schema for creating test source
// if needed can be extended to custom struct
type UpsertTestSourceOptions testkube.TestSourceUpsertRequest

// UpdateTestSourceOptions - is mapping for now to OpenAPI schema for changing test source
// if needed can be extended to custom struct
type UpdateTestSourceOptions testkube.TestSourceUpdateRequest

// CreateTemplateOptions - is mapping for now to OpenAPI schema for creating/changing template
type CreateTemplateOptions testkube.TemplateCreateRequest

// UpdateTemplateOptions - is mapping for now to OpenAPI schema for changing template request
type UpdateTemplateOptions testkube.TemplateUpdateRequest

// TODO consider replacing it with testkube.ExecutionRequest - looks almost the samea and redundant
// ExecuteTestOptions contains test run options
type ExecuteTestOptions struct {
	ExecutionVariables                 map[string]testkube.Variable
	ExecutionVariablesFileContent      string
	IsVariablesFileUploaded            bool
	ExecutionLabels                    map[string]string
	Command                            []string
	Args                               []string
	ArgsMode                           string
	Envs                               map[string]string
	SecretEnvs                         map[string]string
	HTTPProxy                          string
	HTTPSProxy                         string
	Image                              string
	Uploads                            []string
	BucketName                         string
	ArtifactRequest                    *testkube.ArtifactRequest
	JobTemplate                        string
	JobTemplateReference               string
	ContentRequest                     *testkube.TestContentRequest
	PreRunScriptContent                string
	PostRunScriptContent               string
	ExecutePostRunScriptBeforeScraping bool
	SourceScripts                      bool
	ScraperTemplate                    string
	ScraperTemplateReference           string
	PvcTemplate                        string
	PvcTemplateReference               string
	NegativeTest                       bool
	IsNegativeTestChangedOnRun         bool
	EnvConfigMaps                      []testkube.EnvReference
	EnvSecrets                         []testkube.EnvReference
	RunningContext                     *testkube.RunningContext
	SlavePodRequest                    *testkube.PodRequest
	ExecutionNamespace                 string
}

// ExecuteTestSuiteOptions contains test suite run options
type ExecuteTestSuiteOptions struct {
	ExecutionVariables       map[string]testkube.Variable
	HTTPProxy                string
	HTTPSProxy               string
	ExecutionLabels          map[string]string
	ContentRequest           *testkube.TestContentRequest
	RunningContext           *testkube.RunningContext
	ConcurrencyLevel         int32
	JobTemplate              string
	JobTemplateReference     string
	ScraperTemplate          string
	ScraperTemplateReference string
	PvcTemplate              string
	PvcTemplateReference     string
}

// Gettable is an interface of gettable objects
type Gettable interface {
	testkube.Test | testkube.TestSuite | testkube.ExecutorDetails |
		testkube.Webhook | testkube.TestWithExecution | testkube.TestSuiteWithExecution | testkube.TestWithExecutionSummary |
		testkube.TestSuiteWithExecutionSummary | testkube.Artifact | testkube.ServerInfo | testkube.Config | testkube.DebugInfo |
		testkube.TestSource | testkube.Template |
		testkube.TestWorkflow | testkube.TestWorkflowWithExecution | testkube.TestWorkflowTemplate | testkube.TestWorkflowExecution
}

// Executable is an interface of executable objects
type Executable interface {
	testkube.Execution | testkube.TestSuiteExecution | testkube.TestWorkflowExecution |
		testkube.ExecutionsResult | testkube.TestSuiteExecutionsResult | testkube.TestWorkflowExecutionsResult
}

// All is an interface of all objects
type All interface {
	Gettable | Executable
}

// Transport provides methods to execute api calls
type Transport[A All] interface {
	Execute(method, uri string, body []byte, params map[string]string) (result A, err error)
	ExecuteMultiple(method, uri string, body []byte, params map[string]string) (result []A, err error)
	Delete(uri, selector string, isContentExpected bool) error
	ExecuteMethod(method, uri, selector string, isContentExpected bool) error
	GetURI(pathTemplate string, params ...interface{}) string
	GetLogs(uri string, logs chan output.Output) error
	GetLogsV2(uri string, logs chan events.Log) error
	GetTestWorkflowExecutionNotifications(uri string, notifications chan testkube.TestWorkflowExecutionNotification) error
	GetFile(uri, fileName, destination string, params map[string][]string) (name string, err error)
}
