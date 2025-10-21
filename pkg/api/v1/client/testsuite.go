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

// GetTestSuiteWithExecution returns single test suite by id with execution

// ListTestSuites list all test suites

// ListTestSuiteWithExecutionSummaries list all test suite with execution summaries

// CreateTestSuite creates new TestSuite Custom Resource

// UpdateTestSuite updates TestSuite Custom Resource

// DeleteTestSuites deletes all test suites

// DeleteTestSuite deletes single test suite by name

// GetTestSuiteExecution returns test suite execution by excution id
func (c TestSuiteClient) GetTestSuiteExecution(executionID string) (execution testkube.TestSuiteExecution, err error) {
	uri := c.testSuiteExecutionTransport.GetURI("/test-suite-executions/%s", executionID)
	return c.testSuiteExecutionTransport.Execute(http.MethodGet, uri, nil, nil)
}

// AbortTestSuiteExecution aborts a test suite execution

// AbortTestSuiteExecutions aborts all test suite executions

// GetTestSuiteExecutionArtifacts returns test suite execution artifacts by excution id
func (c TestSuiteClient) GetTestSuiteExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error) {
	uri := c.testSuiteArtifactTransport.GetURI("/test-suite-executions/%s/artifacts", executionID)
	return c.testSuiteArtifactTransport.ExecuteMultiple(http.MethodGet, uri, nil, nil)
}

// ExecuteTestSuite starts new external test suite execution, reads data and returns ID
// Execution is started asynchronously client can check later for results

// ExecuteTestSuites starts new external test suite executions, reads data and returns IDs
// Executions are started asynchronously client can check later for results

// WatchTestSuiteExecution watches for changes in channels of test suite executions steps

// ListTestSuiteExecutions list all executions for given test suite
