package client

import (
	"net/http"

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

// GetTestSuiteExecution returns test suite execution by excution id
func (c TestSuiteClient) GetTestSuiteExecution(executionID string) (execution testkube.TestSuiteExecution, err error) {
	uri := c.testSuiteExecutionTransport.GetURI("/test-suite-executions/%s", executionID)
	return c.testSuiteExecutionTransport.Execute(http.MethodGet, uri, nil, nil)
}

// GetTestSuiteExecutionArtifacts returns test suite execution artifacts by excution id
func (c TestSuiteClient) GetTestSuiteExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error) {
	uri := c.testSuiteArtifactTransport.GetURI("/test-suite-executions/%s/artifacts", executionID)
	return c.testSuiteArtifactTransport.ExecuteMultiple(http.MethodGet, uri, nil, nil)
}
