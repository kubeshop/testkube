package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/logs/events"
)

// NewTestClient creates new Test client
func NewTestClient(
	testTransport Transport[testkube.Test],
	executionTransport Transport[testkube.Execution],
	testWithExecutionTransport Transport[testkube.TestWithExecution],
	testWithExecutionSummaryTransport Transport[testkube.TestWithExecutionSummary],
	executionsResultTransport Transport[testkube.ExecutionsResult],
	artifactTransport Transport[testkube.Artifact],
	serverInfoTransport Transport[testkube.ServerInfo],
	debugInfoTransport Transport[testkube.DebugInfo],
) TestClient {
	return TestClient{
		testTransport:                     testTransport,
		executionTransport:                executionTransport,
		testWithExecutionTransport:        testWithExecutionTransport,
		testWithExecutionSummaryTransport: testWithExecutionSummaryTransport,
		executionsResultTransport:         executionsResultTransport,
		artifactTransport:                 artifactTransport,
		serverInfoTransport:               serverInfoTransport,
		debugInfoTransport:                debugInfoTransport,
	}
}

// TestClient is a client for tests
type TestClient struct {
	testTransport                     Transport[testkube.Test]
	executionTransport                Transport[testkube.Execution]
	testWithExecutionTransport        Transport[testkube.TestWithExecution]
	testWithExecutionSummaryTransport Transport[testkube.TestWithExecutionSummary]
	executionsResultTransport         Transport[testkube.ExecutionsResult]
	artifactTransport                 Transport[testkube.Artifact]
	serverInfoTransport               Transport[testkube.ServerInfo]
	debugInfoTransport                Transport[testkube.DebugInfo]
}

// GetTest returns single test by id
func (c TestClient) GetTest(id string) (test testkube.Test, err error) {
	uri := c.testTransport.GetURI("/tests/%s", id)
	return c.testTransport.Execute(http.MethodGet, uri, nil, nil)
}

// GetTestWithExecution returns single test by id with execution
func (c TestClient) GetTestWithExecution(id string) (test testkube.TestWithExecution, err error) {
	uri := c.testWithExecutionTransport.GetURI("/test-with-executions/%s", id)
	return c.testWithExecutionTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListTests list all tests
func (c TestClient) ListTests(selector string) (tests testkube.Tests, err error) {
	uri := c.testTransport.GetURI("/tests")
	params := map[string]string{
		"selector": selector,
	}

	return c.testTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// ListTestWithExecutionSummaries list all test with execution summaries
func (c TestClient) ListTestWithExecutionSummaries(selector string) (testWithExecutionSummaries testkube.TestWithExecutionSummaries, err error) {
	uri := c.testWithExecutionSummaryTransport.GetURI("/test-with-executions")
	params := map[string]string{
		"selector": selector,
	}

	return c.testWithExecutionSummaryTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// CreateTest creates new Test Custom Resource
func (c TestClient) CreateTest(options UpsertTestOptions) (test testkube.Test, err error) {
	uri := c.testTransport.GetURI("/tests")
	request := testkube.TestUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return test, err
	}

	return c.testTransport.Execute(http.MethodPost, uri, body, nil)
}

// UpdateTest updates Test Custom Resource
func (c TestClient) UpdateTest(options UpdateTestOptions) (test testkube.Test, err error) {
	name := ""
	if options.Name != nil {
		name = *options.Name
	}

	uri := c.testTransport.GetURI("/tests/%s", name)
	request := testkube.TestUpdateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return test, err
	}

	return c.testTransport.Execute(http.MethodPatch, uri, body, nil)
}

// DeleteTests deletes all tests
func (c TestClient) DeleteTests(selector string) error {
	uri := c.testTransport.GetURI("/tests")
	return c.testTransport.Delete(uri, selector, true)
}

// DeleteTest deletes single test by name
func (c TestClient) DeleteTest(name string) error {
	if name == "" {
		return fmt.Errorf("test name '%s' is not valid", name)
	}

	uri := c.testTransport.GetURI("/tests/%s", name)
	return c.testTransport.Delete(uri, "", true)
}

// GetExecution returns test execution by excution id
func (c TestClient) GetExecution(executionID string) (execution testkube.Execution, err error) {
	uri := c.executionTransport.GetURI("/executions/%s", executionID)
	return c.executionTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ExecuteTest starts test execution, reads data and returns ID
// execution is started asynchronously client can check later for results
func (c TestClient) ExecuteTest(id, executionName string, options ExecuteTestOptions) (execution testkube.Execution, err error) {
	uri := c.executionTransport.GetURI("/tests/%s/executions", id)
	request := testkube.ExecutionRequest{
		Name:                               executionName,
		IsVariablesFileUploaded:            options.IsVariablesFileUploaded,
		VariablesFile:                      options.ExecutionVariablesFileContent,
		Variables:                          options.ExecutionVariables,
		Envs:                               options.Envs,
		Command:                            options.Command,
		Args:                               options.Args,
		ArgsMode:                           options.ArgsMode,
		SecretEnvs:                         options.SecretEnvs,
		HttpProxy:                          options.HTTPProxy,
		HttpsProxy:                         options.HTTPSProxy,
		ExecutionLabels:                    options.ExecutionLabels,
		Image:                              options.Image,
		Uploads:                            options.Uploads,
		BucketName:                         options.BucketName,
		ArtifactRequest:                    options.ArtifactRequest,
		JobTemplate:                        options.JobTemplate,
		JobTemplateReference:               options.JobTemplateReference,
		ContentRequest:                     options.ContentRequest,
		PreRunScript:                       options.PreRunScriptContent,
		PostRunScript:                      options.PostRunScriptContent,
		ExecutePostRunScriptBeforeScraping: options.ExecutePostRunScriptBeforeScraping,
		SourceScripts:                      options.SourceScripts,
		ScraperTemplate:                    options.ScraperTemplate,
		ScraperTemplateReference:           options.ScraperTemplateReference,
		PvcTemplate:                        options.PvcTemplate,
		PvcTemplateReference:               options.PvcTemplateReference,
		NegativeTest:                       options.NegativeTest,
		IsNegativeTestChangedOnRun:         options.IsNegativeTestChangedOnRun,
		EnvConfigMaps:                      options.EnvConfigMaps,
		EnvSecrets:                         options.EnvSecrets,
		RunningContext:                     options.RunningContext,
		SlavePodRequest:                    options.SlavePodRequest,
		ExecutionNamespace:                 options.ExecutionNamespace,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return execution, err
	}

	return c.executionTransport.Execute(http.MethodPost, uri, body, nil)
}

// ExecuteTests starts test executions, reads data and returns IDs
// executions are started asynchronously client can check later for results
func (c TestClient) ExecuteTests(selector string, concurrencyLevel int, options ExecuteTestOptions) (executions []testkube.Execution, err error) {
	uri := c.executionTransport.GetURI("/executions")
	request := testkube.ExecutionRequest{
		IsVariablesFileUploaded:            options.IsVariablesFileUploaded,
		VariablesFile:                      options.ExecutionVariablesFileContent,
		Variables:                          options.ExecutionVariables,
		Envs:                               options.Envs,
		Command:                            options.Command,
		Args:                               options.Args,
		ArgsMode:                           options.ArgsMode,
		SecretEnvs:                         options.SecretEnvs,
		HttpProxy:                          options.HTTPProxy,
		HttpsProxy:                         options.HTTPSProxy,
		Uploads:                            options.Uploads,
		BucketName:                         options.BucketName,
		ArtifactRequest:                    options.ArtifactRequest,
		JobTemplate:                        options.JobTemplate,
		JobTemplateReference:               options.JobTemplateReference,
		ContentRequest:                     options.ContentRequest,
		PreRunScript:                       options.PreRunScriptContent,
		PostRunScript:                      options.PostRunScriptContent,
		ExecutePostRunScriptBeforeScraping: options.ExecutePostRunScriptBeforeScraping,
		SourceScripts:                      options.SourceScripts,
		ScraperTemplate:                    options.ScraperTemplate,
		ScraperTemplateReference:           options.ScraperTemplateReference,
		PvcTemplate:                        options.PvcTemplate,
		PvcTemplateReference:               options.PvcTemplateReference,
		NegativeTest:                       options.NegativeTest,
		IsNegativeTestChangedOnRun:         options.IsNegativeTestChangedOnRun,
		RunningContext:                     options.RunningContext,
		SlavePodRequest:                    options.SlavePodRequest,
		ExecutionNamespace:                 options.ExecutionNamespace,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return executions, err
	}

	params := map[string]string{
		"selector":    selector,
		"concurrency": strconv.Itoa(concurrencyLevel),
	}

	return c.executionTransport.ExecuteMultiple(http.MethodPost, uri, body, params)
}

// AbortExecution aborts execution by testId and id
func (c TestClient) AbortExecution(testID, id string) error {
	uri := c.executionTransport.GetURI("/tests/%s/executions/%s", testID, id)
	return c.executionTransport.ExecuteMethod(http.MethodPatch, uri, "", false)
}

// AbortExecutions aborts all the executions of a test
func (c TestClient) AbortExecutions(testID string) error {
	uri := c.executionTransport.GetURI("/tests/%s/abort", testID)
	return c.executionTransport.ExecuteMethod(http.MethodPost, uri, "", false)
}

// ListExecutions list all executions for given test name
func (c TestClient) ListExecutions(id string, limit int, selector string) (executions testkube.ExecutionsResult, err error) {
	uri := c.executionsResultTransport.GetURI("/executions/")
	if id != "" {
		uri = c.executionsResultTransport.GetURI(fmt.Sprintf("/tests/%s/executions", id))
	}

	params := map[string]string{
		"selector": selector,
		"pageSize": fmt.Sprintf("%d", limit),
	}

	return c.executionsResultTransport.Execute(http.MethodGet, uri, nil, params)
}

// Logs returns logs stream from job pods, based on job pods logs
func (c TestClient) Logs(id string) (logs chan output.Output, err error) {
	logs = make(chan output.Output)
	uri := c.testTransport.GetURI("/executions/%s/logs", id)
	err = c.testTransport.GetLogs(uri, logs)
	return logs, err
}

// LogsV2 returns logs version 2 stream from log sever, based on job pods logs
func (c TestClient) LogsV2(id string) (logs chan events.Log, err error) {
	logs = make(chan events.Log)
	uri := c.testTransport.GetURI("/executions/%s/logs/v2", id)
	err = c.testTransport.GetLogsV2(uri, logs)
	return logs, err
}

// GetExecutionArtifacts returns execution artifacts
func (c TestClient) GetExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error) {
	uri := c.artifactTransport.GetURI("/executions/%s/artifacts", executionID)
	return c.artifactTransport.ExecuteMultiple(http.MethodGet, uri, nil, nil)
}

// DownloadFile downloads file
func (c TestClient) DownloadFile(executionID, fileName, destination string) (artifact string, err error) {
	uri := c.executionTransport.GetURI("/executions/%s/artifacts/%s", executionID, url.QueryEscape(fileName))
	return c.executionTransport.GetFile(uri, fileName, destination, nil)
}

// DownloadArchive downloads archive
func (c TestClient) DownloadArchive(executionID, destination string, masks []string) (archive string, err error) {
	uri := c.executionTransport.GetURI("/executions/%s/artifact-archive", executionID)
	return c.executionTransport.GetFile(uri, fmt.Sprintf("%s.tar.gz", executionID), destination, map[string][]string{"mask": masks})
}

// GetServerInfo returns server info
func (c TestClient) GetServerInfo() (info testkube.ServerInfo, err error) {
	uri := c.serverInfoTransport.GetURI("/info")
	return c.serverInfoTransport.Execute(http.MethodGet, uri, nil, nil)
}

func (c TestClient) GetDebugInfo() (debugInfo testkube.DebugInfo, err error) {
	uri := c.debugInfoTransport.GetURI("/debug")
	return c.debugInfoTransport.Execute(http.MethodGet, uri, nil, nil)
}
