package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewTestSuiteClient creates new TestSuite client
func NewTestSuiteClient(
	testSuiteTransport Transport[testkube.TestSuite],
	testSuiteExecutionTransport Transport[testkube.TestSuiteExecution],
	testSuiteWithExecutionTransport Transport[testkube.TestSuiteWithExecution],
	testSuiteWithExecutionSummaryTransport Transport[testkube.TestSuiteWithExecutionSummary],
	testSuiteExecutionsResultTransport Transport[testkube.TestSuiteExecutionsResult],
	testSuiteArtifactTransport Transport[testkube.Artifact],
) TestSuiteClient {
	return TestSuiteClient{
		testSuiteTransport:                     testSuiteTransport,
		testSuiteExecutionTransport:            testSuiteExecutionTransport,
		testSuiteWithExecutionTransport:        testSuiteWithExecutionTransport,
		testSuiteWithExecutionSummaryTransport: testSuiteWithExecutionSummaryTransport,
		testSuiteExecutionsResultTransport:     testSuiteExecutionsResultTransport,
		testSuiteArtifactTransport:             testSuiteArtifactTransport,
	}
}

// TestSuiteClient is a client for test suites
type TestSuiteClient struct {
	testSuiteTransport                     Transport[testkube.TestSuite]
	testSuiteExecutionTransport            Transport[testkube.TestSuiteExecution]
	testSuiteWithExecutionTransport        Transport[testkube.TestSuiteWithExecution]
	testSuiteWithExecutionSummaryTransport Transport[testkube.TestSuiteWithExecutionSummary]
	testSuiteExecutionsResultTransport     Transport[testkube.TestSuiteExecutionsResult]
	testSuiteArtifactTransport             Transport[testkube.Artifact]
}

// GetTestSuite returns single test suite by id
func (c TestSuiteClient) GetTestSuite(id string) (testSuite testkube.TestSuite, err error) {
	uri := c.testSuiteTransport.GetURI("/test-suites/%s", id)
	return c.testSuiteTransport.Execute(http.MethodGet, uri, nil, nil)
}

// GetTestSuitWithExecution returns single test suite by id with execution
func (c TestSuiteClient) GetTestSuiteWithExecution(id string) (test testkube.TestSuiteWithExecution, err error) {
	uri := c.testSuiteWithExecutionTransport.GetURI("/test-suite-with-executions/%s", id)
	return c.testSuiteWithExecutionTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListTestSuites list all test suites
func (c TestSuiteClient) ListTestSuites(selector string) (testSuites testkube.TestSuites, err error) {
	uri := c.testSuiteTransport.GetURI("/test-suites")
	params := map[string]string{
		"selector": selector,
	}

	return c.testSuiteTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// ListTestSuiteWithExecutionSummaries list all test suite with execution summaries
func (c TestSuiteClient) ListTestSuiteWithExecutionSummaries(selector string) (
	testSuiteWithExecutionSummaries testkube.TestSuiteWithExecutionSummaries, err error) {
	uri := c.testSuiteWithExecutionSummaryTransport.GetURI("/test-suite-with-executions")
	params := map[string]string{
		"selector": selector,
	}

	return c.testSuiteWithExecutionSummaryTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// CreateTestSuite creates new TestSuite Custom Resource
func (c TestSuiteClient) CreateTestSuite(options UpsertTestSuiteOptions) (testSuite testkube.TestSuite, err error) {
	uri := c.testSuiteTransport.GetURI("/test-suites")
	request := testkube.TestSuiteUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return testSuite, err
	}

	return c.testSuiteTransport.Execute(http.MethodPost, uri, body, nil)
}

// UpdateTestSuite updates TestSuite Custom Resource
func (c TestSuiteClient) UpdateTestSuite(options UpdateTestSuiteOptions) (testSuite testkube.TestSuite, err error) {
	name := ""
	if options.Name != nil {
		name = *options.Name
	}

	uri := c.testSuiteTransport.GetURI("/test-suites/%s", name)
	request := testkube.TestSuiteUpdateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return testSuite, err
	}

	return c.testSuiteTransport.Execute(http.MethodPatch, uri, body, nil)
}

// DeleteTestSuites deletes all test suites
func (c TestSuiteClient) DeleteTestSuites(selector string) error {
	uri := c.testSuiteTransport.GetURI("/test-suites")
	return c.testSuiteTransport.Delete(uri, selector, true)
}

// DeleteTestSuite deletes single test suite by name
func (c TestSuiteClient) DeleteTestSuite(name string) error {
	if name == "" {
		return fmt.Errorf("test suite name '%s' is not valid", name)
	}

	uri := c.testSuiteTransport.GetURI("/test-suites/%s", name)
	return c.testSuiteTransport.Delete(uri, "", true)
}

// GetTestSuiteExecution returns test suite execution by excution id
func (c TestSuiteClient) GetTestSuiteExecution(executionID string) (execution testkube.TestSuiteExecution, err error) {
	uri := c.testSuiteExecutionTransport.GetURI("/test-suite-executions/%s", executionID)
	return c.testSuiteExecutionTransport.Execute(http.MethodGet, uri, nil, nil)
}

// AbortTestSuiteExecution aborts a test suite execution
func (c TestSuiteClient) AbortTestSuiteExecution(executionID string) error {
	uri := c.testSuiteExecutionTransport.GetURI("/test-suite-executions/%s", executionID)
	return c.testSuiteExecutionTransport.ExecuteMethod(http.MethodPatch, uri, "", false)
}

// AbortTestSuiteExecutions aborts all test suite executions
func (c TestSuiteClient) AbortTestSuiteExecutions(testSuiteName string) error {
	uri := c.testSuiteExecutionTransport.GetURI("/test-suites/%s/abort", testSuiteName)
	return c.testSuiteExecutionTransport.ExecuteMethod(http.MethodPost, uri, "", false)
}

// GetTestSuiteExecutionArtifacts returns test suite execution artifacts by excution id
func (c TestSuiteClient) GetTestSuiteExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error) {
	uri := c.testSuiteArtifactTransport.GetURI("/test-suite-executions/%s/artifacts", executionID)
	return c.testSuiteArtifactTransport.ExecuteMultiple(http.MethodGet, uri, nil, nil)
}

// ExecuteTestSuite starts new external test suite execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c TestSuiteClient) ExecuteTestSuite(id, executionName string, options ExecuteTestSuiteOptions) (execution testkube.TestSuiteExecution, err error) {
	uri := c.testSuiteExecutionTransport.GetURI("/test-suites/%s/executions", id)
	executionRequest := testkube.TestSuiteExecutionRequest{
		Name:                     executionName,
		Variables:                options.ExecutionVariables,
		HttpProxy:                options.HTTPProxy,
		HttpsProxy:               options.HTTPSProxy,
		ExecutionLabels:          options.ExecutionLabels,
		ContentRequest:           options.ContentRequest,
		RunningContext:           options.RunningContext,
		ConcurrencyLevel:         options.ConcurrencyLevel,
		JobTemplate:              options.JobTemplate,
		JobTemplateReference:     options.JobTemplateReference,
		ScraperTemplate:          options.ScraperTemplate,
		ScraperTemplateReference: options.ScraperTemplateReference,
		PvcTemplate:              options.PvcTemplate,
		PvcTemplateReference:     options.PvcTemplateReference,
	}

	body, err := json.Marshal(executionRequest)
	if err != nil {
		return execution, err
	}

	return c.testSuiteExecutionTransport.Execute(http.MethodPost, uri, body, nil)
}

// ExecuteTestSuites starts new external test suite executions, reads data and returns IDs
// Executions are started asynchronously client can check later for results
func (c TestSuiteClient) ExecuteTestSuites(selector string, concurrencyLevel int, options ExecuteTestSuiteOptions) (executions []testkube.TestSuiteExecution, err error) {
	uri := c.testSuiteExecutionTransport.GetURI("/test-suite-executions")
	executionRequest := testkube.TestSuiteExecutionRequest{
		Variables:                options.ExecutionVariables,
		HttpProxy:                options.HTTPProxy,
		HttpsProxy:               options.HTTPSProxy,
		ExecutionLabels:          options.ExecutionLabels,
		ContentRequest:           options.ContentRequest,
		RunningContext:           options.RunningContext,
		ConcurrencyLevel:         options.ConcurrencyLevel,
		JobTemplate:              options.JobTemplate,
		JobTemplateReference:     options.JobTemplateReference,
		ScraperTemplate:          options.ScraperTemplate,
		ScraperTemplateReference: options.ScraperTemplateReference,
		PvcTemplate:              options.PvcTemplate,
		PvcTemplateReference:     options.PvcTemplateReference,
	}

	body, err := json.Marshal(executionRequest)
	if err != nil {
		return executions, err
	}

	params := map[string]string{
		"selector":    selector,
		"concurrency": strconv.Itoa(concurrencyLevel),
	}

	return c.testSuiteExecutionTransport.ExecuteMultiple(http.MethodPost, uri, body, params)
}

// WatchTestSuiteExecution watches for changes in channels of test suite executions steps
func (c TestSuiteClient) WatchTestSuiteExecution(executionID string) (respCh chan testkube.WatchTestSuiteExecutionResponse) {
	respCh = make(chan testkube.WatchTestSuiteExecutionResponse)

	go func() {
		defer close(respCh)

		execution, err := c.GetTestSuiteExecution(executionID)
		if err != nil {
			respCh <- testkube.WatchTestSuiteExecutionResponse{
				Error: err,
			}
			return
		}

		respCh <- testkube.WatchTestSuiteExecutionResponse{
			Execution: execution,
		}

		for range time.NewTicker(time.Second).C {
			execution, err = c.GetTestSuiteExecution(executionID)
			if err != nil {
				respCh <- testkube.WatchTestSuiteExecutionResponse{
					Error: err,
				}
				return
			}

			if execution.IsCompleted() {
				respCh <- testkube.WatchTestSuiteExecutionResponse{
					Execution: execution,
				}
				return
			}

			respCh <- testkube.WatchTestSuiteExecutionResponse{
				Execution: execution,
			}
		}
	}()
	return respCh
}

// ListTestSuiteExecutions list all executions for given test suite
func (c TestSuiteClient) ListTestSuiteExecutions(testID string, limit int, selector string) (executions testkube.TestSuiteExecutionsResult, err error) {
	uri := c.testSuiteExecutionsResultTransport.GetURI("/test-suite-executions")
	params := map[string]string{
		"selector": selector,
		"pageSize": fmt.Sprintf("%d", limit),
		"id":       testID,
	}

	return c.testSuiteExecutionsResultTransport.Execute(http.MethodGet, uri, nil, params)
}
