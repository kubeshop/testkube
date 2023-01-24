package data

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/result"
)

type NextExecutionNumberResultRequest struct {
	TestName string `json:"testName"`
}

type NextExecutionNumberResultResponse struct {
	TestNumber int32 `json:"testNumber"`
}

type GetResultRequest struct {
	ID string `json:"id"`
}

type GetResultResponse struct {
	Execution testkube.Execution `json:"execution"`
}

type GetByNameAndTestResultRequest struct {
	Name     string `json:"name"`
	TestName string `json:"testName"`
}

type GetByNameAndTestResultResponse struct {
	Execution testkube.Execution `json:"execution"`
}

type GetLatestByTestResultRequest struct {
	TestName  string `json:"testName"`
	SortField string `json:"sortField"`
}

type GetLatestByTestResultResponse struct {
	Execution testkube.Execution `json:"execution"`
}

type GetLatestByTestsResultRequest struct {
	TestNames []string `json:"testNames"`
	SortField string   `json:"sortField"`
}

type GetLatestByTestsResultResponse struct {
	Executions []testkube.Execution `json:"executions"`
}

type GetExecutionsResultRequest struct {
	Filter *result.FilterImpl `json:"filter"`
}

type GetExecutionsResultResponse struct {
	Executions []testkube.Execution `json:"executions"`
}

type GetExecutionTotalsResultRequest struct {
	Paging bool                 `json:"paging"`
	Filter []*result.FilterImpl `json:"filter"`
}

type GetExecutionTotalsResultResponse struct {
	Result testkube.ExecutionsTotals `json:"result"`
}

type InsertResultRequest struct {
	Result testkube.Execution `json:"result"`
}

type InsertResultResponse struct {
}

type UpdateResultRequest struct {
	Result testkube.Execution `json:"result"`
}

type UpdateResultResponse struct {
}

type UpdateResultInExecutionResultRequest struct {
	ID        string                   `json:"id"`
	Execution testkube.ExecutionResult `json:"execution"`
}

type UpdateResultInExecutionResultResponse struct {
}

type StartExecutionResultRequest struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"startTime"`
}

type StartExecutionResultResponse struct {
}

type EndExecutionResultRequest struct {
	Execution testkube.Execution `json:"execution"`
}

type EndExecutionResultResponse struct {
}

type GetLabelsResultResponse struct {
	Labels map[string][]string `json:"labels"`
}

type DeleteByTestResultRequest struct {
	TestName string `json:"testName"`
}

type DeleteByTestResultResponse struct {
}

type DeleteByTestSuiteResultRequest struct {
	TestSuiteName string `json:"testSuiteName"`
}

type DeleteByTestSuiteResultResponse struct {
}

type DeleteAllResultRequest struct{}

type DeleteAllResultResponse struct{}

type DeleteByTestsResultRequest struct {
	TestNames []string `json:"testNames"`
}

type DeleteByTestsResultResponse struct{}

type DeleteByTestSuitesResultRequest struct {
	TestSuiteNames []string `json:"testSuiteNames"`
}

type DeleteByTestSuitesResultResponse struct{}

type DeleteForAllTestSuitesResultResponse struct {
}

type GetTestMetricsResultRequest struct {
	Name  string `json:"name"`
	Limit int    `json:"limit"`
	Last  int    `json:"last"`
}

type GetTestMetricsResultResponse struct {
	Metrics testkube.ExecutionsMetrics `json:"metrics"`
}
