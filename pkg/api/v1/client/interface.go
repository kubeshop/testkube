package client

import (
	"io"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/runner/output"
)

type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
	Do(req *http.Request) (resp *http.Response, err error)
}

type Client interface {
	GetExecution(scriptID, executionID string) (execution testkube.Execution, err error)
	ListExecutions(scriptID string, limit int) (executions testkube.ExecutionsResult, err error)
	AbortExecution(script string, id string) error

	GetScript(id string) (script testkube.Script, err error)
	CreateScript(options UpsertScriptOptions) (script testkube.Script, err error)
	UpdateScript(options UpsertScriptOptions) (script testkube.Script, err error)
	DeleteScript(name string, namespace string) error
	DeleteScripts(namespace string) error
	ListScripts(namespace string) (scripts testkube.Scripts, err error)
	ExecuteScript(id, namespace, executionName string, executionParams map[string]string) (execution testkube.Execution, err error)
	Logs(id string) (logs chan output.Output, err error)

	CreateExecutor(options CreateExecutorOptions) (executor testkube.ExecutorDetails, err error)
	GetExecutor(name string) (executor testkube.ExecutorDetails, err error)
	ListExecutors() (executors testkube.ExecutorsDetails, err error)
	DeleteExecutor(name string) (err error)

	GetExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error)
	DownloadFile(executionID, fileName, destination string) (artifact string, err error)

	CreateTest(options UpsertTestOptions) (test testkube.Test, err error)
	UpdateTest(options UpsertTestOptions) (script testkube.Test, err error)
	GetTest(id string, namespace string) (script testkube.Test, err error)
	ListTests(namespace string) (scripts testkube.Tests, err error)
	DeleteTest(name string, namespace string) error
	ExecuteTest(id, namespace, executionName string, executionParams map[string]string) (execution testkube.TestExecution, err error)

	GetTestExecution(executionID string) (execution testkube.TestExecution, err error)
	ListTestExecutions(test string, limit int) (executions testkube.ExecutionsResult, err error)

	GetServerInfo() (scripts testkube.ServerInfo, err error)
}

type UpsertTestOptions testkube.TestUpsertRequest

// UpsertScriptOptions - is mapping for now to OpenAPI schema for creating request
// if needed can beextended to custom struct
type UpsertScriptOptions testkube.ScriptUpsertRequest

// CreateExectorOptions - is mapping for now to OpenAPI schema for creating request
type CreateExecutorOptions testkube.ExecutorCreateRequest
