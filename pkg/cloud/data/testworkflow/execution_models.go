package testworkflow

import (
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

type ExecutionGetRequest struct {
	ID string `json:"id"`
}

type ExecutionGetResponse struct {
	WorkflowExecution testkube.TestWorkflowExecution `json:"workflowExecution"`
}

type ExecutionGetByNameAndWorkflowRequest struct {
	Name         string `json:"name"`
	WorkflowName string `json:"workflowName"`
}

type ExecutionGetByNameAndWorkflowResponse struct {
	WorkflowExecution testkube.TestWorkflowExecution `json:"workflowExecution"`
}

type ExecutionGetLatestByWorkflowRequest struct {
	WorkflowName string `json:"workflowName"`
	SortBy       string `json:"sortBy,omitempty"`
}

type ExecutionGetLatestByWorkflowResponse struct {
	WorkflowExecution *testkube.TestWorkflowExecution `json:"workflowExecution"`
}

type ExecutionGetRunningRequest struct {
}

type ExecutionGetRunningResponse struct {
	WorkflowExecutions []testkube.TestWorkflowExecution `json:"workflowExecutions"`
}

type ExecutionGetFinishedRequest struct {
	Filter *testworkflow.FilterImpl `json:"filter"`
}

type ExecutionGetFinishedResponse struct {
	WorkflowExecutions []testkube.TestWorkflowExecution `json:"workflowExecutions"`
}

type ExecutionGetLatestByWorkflowsRequest struct {
	WorkflowNames []string `json:"workflowNames"`
}

type ExecutionGetLatestByWorkflowsResponse struct {
	WorkflowExecutions []testkube.TestWorkflowExecutionSummary `json:"workflowExecutions"`
}

type ExecutionGetExecutionTotalsRequest struct {
	Filter []*testworkflow.FilterImpl `json:"filter"`
}

type ExecutionGetExecutionTotalsResponse struct {
	Totals testkube.ExecutionsTotals `json:"totals"`
}

type ExecutionCountRequest struct {
	Filter *testworkflow.FilterImpl `json:"filter"`
}

type ExecutionCountResponse struct {
	Count int64 `json:"count"`
}

type ExecutionGetExecutionsRequest struct {
	Filter *testworkflow.FilterImpl `json:"filter"`
}

type ExecutionGetExecutionsResponse struct {
	WorkflowExecutions []testkube.TestWorkflowExecution `json:"workflowExecutions"`
}

type ExecutionGetExecutionsSummaryRequest struct {
	Filter *testworkflow.FilterImpl `json:"filter"`
}

type ExecutionGetExecutionsSummaryResponse struct {
	WorkflowExecutions []testkube.TestWorkflowExecutionSummary `json:"workflowExecutions"`
}

type ExecutionGetPreviousFinishedStateRequest struct {
	WorkflowName string
	Date         time.Time
}

type ExecutionGetPreviousFinishedStateResponse struct {
	Result testkube.TestWorkflowStatus
}

type ExecutionInsertRequest struct {
	WorkflowExecution testkube.TestWorkflowExecution `json:"workflowExecution"`
}

type ExecutionInsertResponse struct {
}

type ExecutionUpdateRequest struct {
	WorkflowExecution testkube.TestWorkflowExecution `json:"workflowExecution"`
}

type ExecutionUpdateResponse struct {
}

type ExecutionUpdateResultRequest struct {
	ID     string                       `json:"id"`
	Result *testkube.TestWorkflowResult `json:"result"`
}

type ExecutionUpdateResultResponse struct {
}

type ExecutionUpdateOutputRequest struct {
	ID     string                        `json:"id"`
	Output []testkube.TestWorkflowOutput `json:"output"`
}

type ExecutionUpdateOutputResponse struct {
}

type ExecutionDeleteByWorkflowRequest struct {
	WorkflowName string `json:"workflowName"`
}

type ExecutionDeleteByWorkflowResponse struct {
}

type ExecutionDeleteOutputByWorkflowRequest struct {
	WorkflowName string `json:"workflowName"`
}

type ExecutionDeleteOutputByWorkflowResponse struct {
}

type ExecutionDeleteOutputForTestWorkflowsRequest struct {
	WorkflowNames []string `json:"workflowNames"`
}

type ExecutionDeleteOutputForTestWorkflowsResponse struct {
}

type ExecutionDeleteAllRequest struct {
}

type ExecutionDeleteAllResponse struct {
}

type ExecutionDeleteByWorkflowsRequest struct {
	WorkflowNames []string `json:"workflowNames"`
}

type ExecutionDeleteByWorkflowsResponse struct {
}

type ExecutionGetWorkflowMetricsRequest struct {
	Name  string `json:"name"`
	Limit int    `json:"limit"`
	Last  int    `json:"last"`
}

type ExecutionGetWorkflowMetricsResponse struct {
	Metrics testkube.ExecutionsMetrics `json:"metrics"`
}

type ExecutionsInsertReportRequest struct {
	ID     string                       `json:"id"`
	Report *testkube.TestWorkflowReport `json:"report"`
}

type ExecutionsInsertReportResponse struct{}

type ExecutionsAddReportRequest struct {
	ID           string `json:"id"`
	WorkflowName string `json:"workflowName"`
	WorkflowStep string `json:"workflowStep"`
	Filepath     string `json:"filepath"`
	Report       []byte `json:"report"`
}

type ExecutionsAddReportResponse struct{}

type ExecutionGetNextExecutionNumberRequest struct {
	TestWorkflowName string `json:"testWorkflowName"`
}

type ExecutionGetNextExecutionNumberResponse struct {
	TestWorkflowNumber int32 `json:"testWorkflowNumber"`
}

type ExecutionGetExecutionTagsRequest struct {
	TestWorkflowName string `json:"testWorkflowName"`
}

type ExecutionGetExecutionTagsResponse struct {
	Tags map[string][]string `json:"tags"`
}

type TestWorkflowListRequest struct {
	Selector string `json:"selector"`
}

type TestWorkflowListResponse struct {
	TestWorkflows []testkube.TestWorkflow `json:"testWorkflows"`
}

type TestWorkflowGetRequest struct {
	Name string `json:"name"`
}

type TestWorkflowGetResponse struct {
	TestWorkflow testkube.TestWorkflow `json:"testWorkflow"`
}

type TestWorkflowTemplateGetRequest struct {
	Name string `json:"name"`
}

type TestWorkflowTemplateGetResponse struct {
	TestWorkflowTemplate testkube.TestWorkflowTemplate `json:"testWorkflowTemplate"`
}
