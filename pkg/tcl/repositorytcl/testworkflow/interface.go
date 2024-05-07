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
	"io"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const PageDefaultLimit int = 100

type Filter interface {
	Name() string
	NameDefined() bool
	LastNDays() int
	LastNDaysDefined() bool
	StartDate() time.Time
	StartDateDefined() bool
	EndDate() time.Time
	EndDateDefined() bool
	Statuses() []testkube.TestWorkflowStatus
	StatusesDefined() bool
	Page() int
	PageSize() int
	TextSearchDefined() bool
	TextSearch() string
	Selector() string
}

//go:generate mockgen -destination=./mock_repository.go -package=testworkflow "github.com/kubeshop/testkube/pkg/tcl/repositorytcl/testworkflow" Repository
type Repository interface {
	// Get gets execution result by id or name
	Get(ctx context.Context, id string) (testkube.TestWorkflowExecution, error)
	// GetByNameAndTestWorkflow gets execution result by name
	GetByNameAndTestWorkflow(ctx context.Context, name, workflowName string) (testkube.TestWorkflowExecution, error)
	// GetLatestByTestWorkflow gets latest execution result by workflow
	GetLatestByTestWorkflow(ctx context.Context, workflowName string) (*testkube.TestWorkflowExecution, error)
	// GetRunning get list of executions that are still running
	GetRunning(ctx context.Context) ([]testkube.TestWorkflowExecution, error)
	// GetLatestByTestWorkflows gets latest execution results by workflow names
	GetLatestByTestWorkflows(ctx context.Context, workflowNames []string) (executions []testkube.TestWorkflowExecutionSummary, err error)
	// GetExecutionsTotals gets executions total stats using a filter, use filter with no data for all
	GetExecutionsTotals(ctx context.Context, filter ...Filter) (totals testkube.ExecutionsTotals, err error)
	// GetExecutions gets executions using a filter, use filter with no data for all
	GetExecutions(ctx context.Context, filter Filter) ([]testkube.TestWorkflowExecution, error)
	// GetExecutionsSummary gets executions summary using a filter, use filter with no data for all
	GetExecutionsSummary(ctx context.Context, filter Filter) ([]testkube.TestWorkflowExecutionSummary, error)
	// Insert inserts new execution result
	Insert(ctx context.Context, result testkube.TestWorkflowExecution) error
	// Update updates execution
	Update(ctx context.Context, result testkube.TestWorkflowExecution) error
	// UpdateResult updates execution result
	UpdateResult(ctx context.Context, id string, result *testkube.TestWorkflowResult) (err error)
	// UpdateReport appends a report to the execution
	UpdateReport(ctx context.Context, id string, report *testkube.TestWorkflowReport) (err error)
	// UpdateOutput updates list of output references in the execution result
	UpdateOutput(ctx context.Context, id string, output []testkube.TestWorkflowOutput) (err error)
	// DeleteByTestWorkflow deletes execution results by workflow
	DeleteByTestWorkflow(ctx context.Context, workflowName string) error
	// DeleteAll deletes all execution results
	DeleteAll(ctx context.Context) error
	// DeleteByTestWorkflows deletes execution results by workflows
	DeleteByTestWorkflows(ctx context.Context, workflowNames []string) (err error)
	// GetTestWorkflowMetrics get metrics based on the TestWorkflow results
	GetTestWorkflowMetrics(ctx context.Context, name string, limit, last int) (metrics testkube.ExecutionsMetrics, err error)
}

//go:generate mockgen -destination=./mock_output_repository.go -package=testworkflow "github.com/kubeshop/testkube/pkg/tcl/repositorytcl/testworkflow" OutputRepository
type OutputRepository interface {
	// PresignSaveLog builds presigned storage URL to save the output in Minio
	PresignSaveLog(ctx context.Context, id, workflowName string) (string, error)
	// PresignReadLog builds presigned storage URL to read the output from Minio
	PresignReadLog(ctx context.Context, id, workflowName string) (string, error)
	// SaveLog streams the output from the workflow to Minio
	SaveLog(ctx context.Context, id, workflowName string, reader io.Reader) error
	// ReadLog streams the output from Minio
	ReadLog(ctx context.Context, id, workflowName string) (io.Reader, error)
	// HasLog checks if there is an output in Minio
	HasLog(ctx context.Context, id, workflowName string) (bool, error)
}
