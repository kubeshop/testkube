package result

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// TODO: Adjust when it gets too small.
const PageDefaultLimit int = 1000

type Filter interface {
	ScriptName() string
	ScriptNameDefined() bool
	StartDate() time.Time
	StartDateDefined() bool
	EndDate() time.Time
	EndDateDefined() bool
	Status() testkube.ExecutionStatus
	StatusDefined() bool
	Page() int
	PageSize() int
	TextSearchDefined() bool
	TextSearch() string
	Tags() []string
}

type Repository interface {
	// Get gets execution result by id
	Get(ctx context.Context, id string) (testkube.Execution, error)
	// GetByNameAndScript gets execution result by name
	GetByNameAndScript(ctx context.Context, name, script string) (testkube.Execution, error)
	// GetExecutions gets executions using a filter, use filter with no data for all
	GetExecutions(ctx context.Context, filter Filter) ([]testkube.Execution, error)
	// GetExecutionTotals gets the statistics on number of executions using a filter, use filter with no data for all
	GetExecutionTotals(ctx context.Context, filter Filter) (result testkube.ExecutionsTotals, err error)
	// Insert inserts new execution result
	Insert(ctx context.Context, result testkube.Execution) error
	// Update updates execution result
	Update(ctx context.Context, result testkube.Execution) error
	// UpdateExecution updates result in execution
	UpdateResult(ctx context.Context, id string, execution testkube.ExecutionResult) error
	// StartExecution updates execution start time
	StartExecution(ctx context.Context, id string, startTime time.Time) error
	// EndExecution updates execution end time
	EndExecution(ctx context.Context, id string, endTime time.Time) error
	// GetTags get all available tags
	GetTags(ctx context.Context) (tags []string, err error)
}
