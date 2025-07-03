package testworkflow

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	testworkflow2 "github.com/kubeshop/testkube/pkg/repository/testworkflow"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
)

var _ testworkflow2.Repository = (*CloudRepository)(nil)

type CloudRepository struct {
	executor executor.Executor
}

func NewCloudRepository(client cloud.TestKubeCloudAPIClient, apiKey string) *CloudRepository {
	return &CloudRepository{executor: executor.NewCloudGRPCExecutor(client, apiKey)}
}

func (r *CloudRepository) Get(ctx context.Context, id string) (testkube.TestWorkflowExecution, error) {
	req := ExecutionGetRequest{ID: id}
	process := func(v ExecutionGetResponse) testkube.TestWorkflowExecution {
		return v.WorkflowExecution
	}
	return pass(r.executor, ctx, req, process)
}

func (r *CloudRepository) GetByNameAndTestWorkflow(ctx context.Context, name, workflowName string) (result testkube.TestWorkflowExecution, err error) {
	req := ExecutionGetByNameAndWorkflowRequest{Name: name, WorkflowName: workflowName}
	process := func(v ExecutionGetResponse) testkube.TestWorkflowExecution {
		return v.WorkflowExecution
	}
	return pass(r.executor, ctx, req, process)
}

func (r *CloudRepository) GetLatestByTestWorkflow(ctx context.Context, workflowName string) (*testkube.TestWorkflowExecution, error) {
	req := ExecutionGetLatestByWorkflowRequest{WorkflowName: workflowName}
	process := func(v ExecutionGetLatestByWorkflowResponse) *testkube.TestWorkflowExecution {
		return v.WorkflowExecution
	}
	return pass(r.executor, ctx, req, process)
}

func (r *CloudRepository) GetLatestByTestWorkflows(ctx context.Context, workflowNames []string) (executions []testkube.TestWorkflowExecutionSummary, err error) {
	req := ExecutionGetLatestByWorkflowsRequest{WorkflowNames: workflowNames}
	process := func(v ExecutionGetLatestByWorkflowsResponse) []testkube.TestWorkflowExecutionSummary {
		return v.WorkflowExecutions
	}
	return pass(r.executor, ctx, req, process)
}

func (r *CloudRepository) GetRunning(ctx context.Context) (result []testkube.TestWorkflowExecution, err error) {
	req := ExecutionGetRunningRequest{}
	process := func(v ExecutionGetRunningResponse) []testkube.TestWorkflowExecution {
		return v.WorkflowExecutions
	}
	return pass(r.executor, ctx, req, process)
}

func (r *CloudRepository) GetFinished(ctx context.Context, filter testworkflow2.Filter) (result []testkube.TestWorkflowExecution, err error) {
	req := ExecutionGetFinishedRequest{Filter: filter.(*testworkflow2.FilterImpl)}
	process := func(v ExecutionGetFinishedResponse) []testkube.TestWorkflowExecution {
		return v.WorkflowExecutions
	}
	return pass(r.executor, ctx, req, process)
}

func (r *CloudRepository) GetExecutionsTotals(ctx context.Context, filter ...testworkflow2.Filter) (totals testkube.ExecutionsTotals, err error) {
	req := ExecutionGetExecutionTotalsRequest{Filter: mapFilters(filter)}
	process := func(v ExecutionGetExecutionTotalsResponse) testkube.ExecutionsTotals {
		return v.Totals
	}
	return pass(r.executor, ctx, req, process)
}

func (r *CloudRepository) GetExecutions(ctx context.Context, filter testworkflow2.Filter) (result []testkube.TestWorkflowExecution, err error) {
	req := ExecutionGetExecutionsRequest{Filter: filter.(*testworkflow2.FilterImpl)}
	process := func(v ExecutionGetExecutionsResponse) []testkube.TestWorkflowExecution {
		return v.WorkflowExecutions
	}
	return pass(r.executor, ctx, req, process)
}

func (r *CloudRepository) GetExecutionsSummary(ctx context.Context, filter testworkflow2.Filter) (result []testkube.TestWorkflowExecutionSummary, err error) {
	req := ExecutionGetExecutionsSummaryRequest{Filter: filter.(*testworkflow2.FilterImpl)}
	process := func(v ExecutionGetExecutionsSummaryResponse) []testkube.TestWorkflowExecutionSummary {
		return v.WorkflowExecutions
	}
	return pass(r.executor, ctx, req, process)
}

func (r *CloudRepository) Insert(ctx context.Context, result testkube.TestWorkflowExecution) (err error) {
	req := ExecutionInsertRequest{WorkflowExecution: result}
	return passNoContent(r.executor, ctx, req)
}

func (r *CloudRepository) Update(ctx context.Context, result testkube.TestWorkflowExecution) (err error) {
	req := ExecutionUpdateRequest{WorkflowExecution: result}
	return passNoContent(r.executor, ctx, req)
}

// NOTE: this is sending an update to the workflow over grpc
func (r *CloudRepository) UpdateResult(ctx context.Context, id string, result *testkube.TestWorkflowResult) (err error) {
	req := ExecutionUpdateResultRequest{ID: id, Result: result}
	return passNoContent(r.executor, ctx, req)
}

func (r *CloudRepository) UpdateReport(ctx context.Context, id string, report *testkube.TestWorkflowReport) (err error) {
	req := ExecutionsInsertReportRequest{
		ID:     id,
		Report: report,
	}
	return passNoContent(r.executor, ctx, req)
}

func (r *CloudRepository) UpdateOutput(ctx context.Context, id string, output []testkube.TestWorkflowOutput) (err error) {
	req := ExecutionUpdateOutputRequest{ID: id, Output: output}
	return passNoContent(r.executor, ctx, req)
}

// DeleteByTestWorkflow deletes execution results by workflow
func (r *CloudRepository) DeleteByTestWorkflow(ctx context.Context, workflowName string) (err error) {
	req := ExecutionDeleteByWorkflowRequest{WorkflowName: workflowName}
	return passNoContent(r.executor, ctx, req)
}

// DeleteAll deletes all execution results
func (r *CloudRepository) DeleteAll(ctx context.Context) (err error) {
	req := ExecutionDeleteAllRequest{}
	return passNoContent(r.executor, ctx, req)
}

// DeleteByTestWorkflows deletes execution results by workflows
func (r *CloudRepository) DeleteByTestWorkflows(ctx context.Context, workflowNames []string) (err error) {
	req := ExecutionDeleteByWorkflowsRequest{WorkflowNames: workflowNames}
	return passNoContent(r.executor, ctx, req)
}

// GetTestWorkflowMetrics returns test executions metrics
func (r *CloudRepository) GetTestWorkflowMetrics(ctx context.Context, name string, limit, last int) (metrics testkube.ExecutionsMetrics, err error) {
	req := ExecutionGetWorkflowMetricsRequest{Name: name, Limit: limit, Last: last}
	process := func(v ExecutionGetWorkflowMetricsResponse) testkube.ExecutionsMetrics {
		return v.Metrics
	}
	return pass(r.executor, ctx, req, process)
}

// GetPreviousFinishedState gets previous finished execution state by test
func (r *CloudRepository) GetPreviousFinishedState(ctx context.Context, workflowName string, date time.Time) (testkube.TestWorkflowStatus, error) {
	req := ExecutionGetPreviousFinishedStateRequest{WorkflowName: workflowName, Date: date}
	response, err := r.executor.Execute(ctx, CmdTestWorkflowExecutionGetPreviousFinishedState, req)
	if err != nil {
		return "", err
	}
	var commandResponse ExecutionGetPreviousFinishedStateResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return "", err
	}
	return commandResponse.Result, nil
}

func (r *CloudRepository) GetNextExecutionNumber(ctx context.Context, testWorkflowName string) (number int32, err error) {
	req := ExecutionGetNextExecutionNumberRequest{TestWorkflowName: testWorkflowName}
	response, err := r.executor.Execute(ctx, CmdTestWorkflowExecutionGetNextExecutionNumber, req)
	if err != nil {
		return 0, err
	}
	var commandResponse ExecutionGetNextExecutionNumberResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return 0, err
	}
	return commandResponse.TestWorkflowNumber, nil
}

func (r *CloudRepository) GetExecutionTags(ctx context.Context, testWorkflowName string) (tags map[string][]string, err error) {
	req := ExecutionGetExecutionTagsRequest{TestWorkflowName: testWorkflowName}
	process := func(v ExecutionGetExecutionTagsResponse) map[string][]string {
		return v.Tags
	}
	return pass(r.executor, ctx, req, process)
}

// Init sets the initialization data from the runner
// Prefer scheduling directly with TestKubeCloudAPI/ScheduleExecution operation.
// This one is a workaround for older Control Planes. It's not recommended, as it may cause race conditions.
func (r *CloudRepository) Init(ctx context.Context, id string, data testworkflow2.InitData) (err error) {
	execution, err := r.Get(ctx, id)
	if err != nil {
		return
	}
	execution.Namespace = data.Namespace
	execution.Signature = data.Signature
	execution.RunnerId = data.RunnerID
	execution.AssignedAt = data.AssignedAt
	if execution.AssignedAt.IsZero() {
		execution.AssignedAt = time.Now()
	}
	return r.Update(ctx, execution)
}

func (r *CloudRepository) Assign(ctx context.Context, id string, prevRunnerId string, newRunnerId string, assignedAt *time.Time) (bool, error) {
	return false, errors.New("not supported")
}

func (r *CloudRepository) GetUnassigned(ctx context.Context) (result []testkube.TestWorkflowExecution, err error) {
	return nil, errors.New("not supported")
}

func (r *CloudRepository) AbortIfQueued(ctx context.Context, id string) (bool, error) {
	return false, errors.New("not supported")
}

func (r *CloudRepository) UpdateResourceAggregations(ctx context.Context, id string, resourceAggregations *testkube.TestWorkflowExecutionResourceAggregationsReport) error {
	return errors.New("not supported")
}
