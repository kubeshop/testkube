package testworkflow

import (
	"context"
	"io"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type Label struct {
	Key   string
	Value *string
	// If value is nil, we check if key exists / not exists
	Exists *bool
}

type LabelSelector struct {
	Or []Label
}

type InitData struct {
	RunnerID  string
	Namespace string
	Signature []testkube.TestWorkflowSignature
}

const PageDefaultLimit int = 100

type Filter interface {
	Name() string
	NameDefined() bool
	Names() []string
	NamesDefined() bool
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
	TagSelector() string
	LabelSelector() *LabelSelector
	ActorName() string
	ActorNameDefined() bool
	ActorType() testkube.TestWorkflowRunningContextActorType
	ActorTypeDefined() bool
	GroupID() string
	GroupIDDefined() bool
}

//go:generate mockgen -destination=./mock_repository.go -package=testworkflow "github.com/kubeshop/testkube/pkg/repository/testworkflow" Repository
type Repository interface {
	Sequences
	// Get gets execution result by id or name
	Get(ctx context.Context, id string) (testkube.TestWorkflowExecution, error)
	// GetByNameAndTestWorkflow gets execution result by name
	GetByNameAndTestWorkflow(ctx context.Context, name, workflowName string) (testkube.TestWorkflowExecution, error)
	// GetLatestByTestWorkflow gets latest execution result by workflow
	GetLatestByTestWorkflow(ctx context.Context, workflowName string) (*testkube.TestWorkflowExecution, error)
	// GetRunning get list of executions that are still running
	GetRunning(ctx context.Context) ([]testkube.TestWorkflowExecution, error)
	// GetUnassigned get list of executions that is waiting to be executed
	GetUnassigned(ctx context.Context) ([]testkube.TestWorkflowExecution, error)
	// GetLatestByTestWorkflows gets latest execution results by workflow names
	GetLatestByTestWorkflows(ctx context.Context, workflowNames []string) (executions []testkube.TestWorkflowExecutionSummary, err error)
	// GetExecutionsTotals gets executions total stats using a filter, use filter with no data for all
	GetExecutionsTotals(ctx context.Context, filter ...Filter) (totals testkube.ExecutionsTotals, err error)
	// GetExecutions gets executions using a filter, use filter with no data for all
	GetExecutions(ctx context.Context, filter Filter) ([]testkube.TestWorkflowExecution, error)
	// GetExecutionsSummary gets executions summary using a filter, use filter with no data for all
	GetExecutionsSummary(ctx context.Context, filter Filter) ([]testkube.TestWorkflowExecutionSummary, error)
	// GetPreviousFinishedState gets previous finished execution state by test
	GetPreviousFinishedState(ctx context.Context, testName string, date time.Time) (testkube.TestWorkflowStatus, error)
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
	// GetExecutionTags gets execution tags
	GetExecutionTags(ctx context.Context, testWorkflowName string) (map[string][]string, error)
	// Init sets the initialization data from the runner
	Init(ctx context.Context, id string, data InitData) error
	// Assign execution to selected runner
	Assign(ctx context.Context, id string, prevRunnerId string, newRunnerId string) (bool, error)
	// AbortIfQueued marks execution as aborted if it's queued
	AbortIfQueued(ctx context.Context, id string) (bool, error)
}

type Sequences interface {
	// GetNextExecutionNumber gets next execution number by name
	GetNextExecutionNumber(ctx context.Context, name string) (number int32, err error)
}

//go:generate mockgen -destination=./mock_output_repository.go -package=testworkflow "github.com/kubeshop/testkube/pkg/repository/testworkflow" OutputRepository
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

	// DeleteOutputByTestWorkflow deletes execution output by test workflow
	DeleteOutputByTestWorkflow(ctx context.Context, testWorkflowName string) error
	// DeleteOutputForTestWorkflows deletes execution output by test workflows
	DeleteOutputForTestWorkflows(ctx context.Context, workflowNames []string) error
}
