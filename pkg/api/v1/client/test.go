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

// GetTestWithExecution returns single test by id with execution
func (c TestClient) GetTestWithExecution(id string) (test testkube.TestWithExecution, err error) {
	uri := c.testWithExecutionTransport.GetURI("/test-with-executions/%s", id)
	return c.testWithExecutionTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListTests list all tests

// ListTestWithExecutionSummaries list all test with execution summaries

// CreateTest creates new Test Custom Resource

// UpdateTest updates Test Custom Resource

// DeleteTests deletes all tests

// DeleteTest deletes single test by name

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
		DisableWebhooks:                    options.DisableWebhooks,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return execution, err
	}

	return c.executionTransport.Execute(http.MethodPost, uri, body, nil)
}

// ExecuteTests starts test executions, reads data and returns IDs
// executions are started asynchronously client can check later for results

// AbortExecution aborts execution by testId and id

// AbortExecutions aborts all the executions of a test

// ListExecutions list all executions for given test name

// Logs returns logs stream from job pods, based on job pods logs

// LogsV2 returns logs version 2 stream from log sever, based on job pods logs

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

// GetServerInfo returns server info
func (c TestClient) GetServerInfo() (info testkube.ServerInfo, err error) {
	uri := c.serverInfoTransport.GetURI("/info")
	return c.serverInfoTransport.Execute(http.MethodGet, uri, nil, nil)
}

func (c TestClient) GetDebugInfo() (debugInfo testkube.DebugInfo, err error) {
	uri := c.debugInfoTransport.GetURI("/debug")
	return c.debugInfoTransport.Execute(http.MethodGet, uri, nil, nil)
}
