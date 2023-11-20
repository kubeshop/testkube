package testresult

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
)

type GetRequest struct {
	ID string `json:"id"`
}

type GetResponse struct {
	TestSuiteExecution testkube.TestSuiteExecution `json:"testSuiteExecution"`
}

type GetByNameAndTestSuiteRequest struct {
	Name          string `json:"name"`
	TestSuiteName string `json:"testSuiteName"`
}

type GetByNameAndTestSuiteResponse struct {
	TestSuiteExecution testkube.TestSuiteExecution `json:"testSuiteExecution"`
}

type GetLatestByTestSuiteRequest struct {
	TestSuiteName string `json:"testSuiteName"`
	SortField     string `json:"sortField"`
}

type GetLatestByTestSuiteResponse struct {
	TestSuiteExecution testkube.TestSuiteExecution `json:"testSuiteExecution"`
}

type GetLatestByTestSuitesRequest struct {
	TestSuiteNames []string `json:"testSuiteNames"`
	SortField      string   `json:"sortField"`
}

type GetLatestByTestSuitesResponse struct {
	TestSuiteExecutions []testkube.TestSuiteExecution `json:"testSuiteExecutions"`
}

type GetExecutionsTotalsRequest struct {
	Filter []*testresult.FilterImpl `json:"filter"`
}

type GetExecutionsTotalsResponse struct {
	ExecutionsTotals testkube.ExecutionsTotals `json:"executionsTotals"`
}

type GetExecutionsRequest struct {
	Filter *testresult.FilterImpl `json:"filter"`
}

type GetExecutionsResponse struct {
	TestSuiteExecutions []testkube.TestSuiteExecution `json:"testSuiteExecutions"`
}

type InsertRequest struct {
	TestSuiteExecution testkube.TestSuiteExecution `json:"testSuiteExecution"`
}

type InsertResponse struct{}

type UpdateRequest struct {
	TestSuiteExecution testkube.TestSuiteExecution `json:"testSuiteExecution"`
}

type UpdateResponse struct{}

type StartExecutionRequest struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"startTime"`
}

type StartExecutionResponse struct {
}

type EndExecutionRequest struct {
	Execution testkube.TestSuiteExecution `json:"execution"`
}

type EndExecutionResponse struct{}

type DeleteByTestSuiteRequest struct {
	TestSuiteName string `json:"testSuiteName"`
}

type DeleteByTestSuiteResponse struct{}

type DeleteAllTestResultsRequest struct{}

type DeleteAllTestResultsResponse struct{}

type DeleteByTestSuitesRequest struct {
	TestSuiteNames []string `json:"testSuiteNames"`
}

type DeleteByTestSuitesResponse struct{}

type GetTestSuiteMetricsRequest struct {
	Name  string `json:"name"`
	Limit int    `json:"limit"`
	Last  int    `json:"last"`
}

type GetTestSuiteMetricsResponse struct {
	Metrics testkube.ExecutionsMetrics `json:"metrics"`
}
