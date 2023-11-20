package result

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/result"
)

type NextExecutionNumberRequest struct {
	TestName string `json:"testName"`
}

type NextExecutionNumberResponse struct {
	TestNumber int32 `json:"testNumber"`
}

type GetRequest struct {
	ID string `json:"id"`
}

type GetResponse struct {
	Execution testkube.Execution `json:"execution"`
}

type GetByNameAndTestRequest struct {
	Name     string `json:"name"`
	TestName string `json:"testName"`
}

type GetByNameAndTestResponse struct {
	Execution testkube.Execution `json:"execution"`
}

type GetLatestByTestRequest struct {
	TestName  string `json:"testName"`
	SortField string `json:"sortField"`
}

type GetLatestByTestResponse struct {
	Execution testkube.Execution `json:"execution"`
}

type GetLatestByTestsRequest struct {
	TestNames []string `json:"testNames"`
	SortField string   `json:"sortField"`
}

type GetLatestByTestsResponse struct {
	Executions []testkube.Execution `json:"executions"`
}

type GetExecutionsRequest struct {
	Filter *result.FilterImpl `json:"filter"`
}

type GetExecutionsResponse struct {
	Executions []testkube.Execution `json:"executions"`
}

type GetExecutionTotalsRequest struct {
	Paging bool                 `json:"paging"`
	Filter []*result.FilterImpl `json:"filter"`
}

type GetExecutionTotalsResponse struct {
	Result testkube.ExecutionsTotals `json:"result"`
}

type InsertRequest struct {
	Result testkube.Execution `json:"result"`
}

type InsertResponse struct {
}

type UpdateRequest struct {
	Result testkube.Execution `json:"result"`
}

type UpdateResponse struct {
}

type UpdateResultInExecutionRequest struct {
	ID        string             `json:"id"`
	Execution testkube.Execution `json:"execution"`
}

type UpdateResultInExecutionResponse struct {
}

type StartExecutionRequest struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"startTime"`
}

type StartExecutionResponse struct {
}

type EndExecutionRequest struct {
	Execution testkube.Execution `json:"execution"`
}

type EndExecutionResponse struct {
}

type GetLabelsResponse struct {
	Labels map[string][]string `json:"labels"`
}

type DeleteByTestRequest struct {
	TestName string `json:"testName"`
}

type DeleteByTestResponse struct {
}

type DeleteByTestSuiteRequest struct {
	TestSuiteName string `json:"testSuiteName"`
}

type DeleteByTestSuiteResponse struct {
}

type DeleteAllRequest struct{}

type DeleteAllResponse struct{}

type DeleteByTestsRequest struct {
	TestNames []string `json:"testNames"`
}

type DeleteByTestsResponse struct{}

type DeleteByTestSuitesRequest struct {
	TestSuiteNames []string `json:"testSuiteNames"`
}

type DeleteByTestSuitesResponse struct{}

type DeleteForAllTestSuitesResponse struct {
}

type GetTestMetricsRequest struct {
	Name  string `json:"name"`
	Limit int    `json:"limit"`
	Last  int    `json:"last"`
}

type GetTestMetricsResponse struct {
	Metrics testkube.ExecutionsMetrics `json:"metrics"`
}
