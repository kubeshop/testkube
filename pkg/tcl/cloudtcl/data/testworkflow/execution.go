// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflow

import (
	"context"

	"github.com/kubeshop/testkube/pkg/tcl/repositorytcl/testworkflow"

	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
)

var _ testworkflow.Repository = (*CloudRepository)(nil)

type CloudRepository struct {
	executor executor.Executor
}

func NewCloudRepository(client cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn, apiKey string) *CloudRepository {
	return &CloudRepository{executor: executor.NewCloudGRPCExecutor(client, grpcConn, apiKey)}
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

func (r *CloudRepository) GetExecutionsTotals(ctx context.Context, filter ...testworkflow.Filter) (totals testkube.ExecutionsTotals, err error) {
	req := ExecutionGetExecutionTotalsRequest{Filter: mapFilters(filter)}
	process := func(v ExecutionGetExecutionTotalsResponse) testkube.ExecutionsTotals {
		return v.Totals
	}
	return pass(r.executor, ctx, req, process)
}

func (r *CloudRepository) GetExecutions(ctx context.Context, filter testworkflow.Filter) (result []testkube.TestWorkflowExecution, err error) {
	req := ExecutionGetExecutionsRequest{Filter: filter.(*testworkflow.FilterImpl)}
	process := func(v ExecutionGetExecutionsResponse) []testkube.TestWorkflowExecution {
		return v.WorkflowExecutions
	}
	return pass(r.executor, ctx, req, process)
}

func (r *CloudRepository) GetExecutionsSummary(ctx context.Context, filter testworkflow.Filter) (result []testkube.TestWorkflowExecutionSummary, err error) {
	req := ExecutionGetExecutionsSummaryRequest{Filter: filter.(*testworkflow.FilterImpl)}
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
