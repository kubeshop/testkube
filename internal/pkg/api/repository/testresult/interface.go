package testresult

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// TODO: Adjust when it gets too small.
const PageDefaultLimit int = 1000

type Filter interface {
	Name() string
	NameDefined() bool
	StartDate() time.Time
	StartDateDefined() bool
	EndDate() time.Time
	EndDateDefined() bool
	Page() int
	PageSize() int
	TextSearchDefined() bool
	TextSearch() string
	Selector() string
}

type Repository interface {
	// Get gets execution result by id
	Get(ctx context.Context, id string) (testkube.TestSuiteExecution, error)
	// GetByNameAndTest gets execution result by name
	GetByNameAndTest(ctx context.Context, name, testName string) (testkube.TestSuiteExecution, error)
	// GetLatestByTest gets latest execution result by test
	GetLatestByTest(ctx context.Context, testName string) (testkube.TestSuiteExecution, error)
	// GetLatestByTests gets latest execution results by test names
	GetLatestByTests(ctx context.Context, testNames []string) (executions []testkube.TestSuiteExecution, err error)
	// GetExecutionsTotals gets executions total stats using a filter, use filter with no data for all
	GetExecutionsTotals(ctx context.Context, filter ...Filter) (totals testkube.ExecutionsTotals, err error)
	// GetExecutions gets executions using a filter, use filter with no data for all
	GetExecutions(ctx context.Context, filter Filter) ([]testkube.TestSuiteExecution, error)
	// Insert inserts new execution result
	Insert(ctx context.Context, result testkube.TestSuiteExecution) error
	// Update updates execution result
	Update(ctx context.Context, result testkube.TestSuiteExecution) error
	// StartExecution updates execution start time
	StartExecution(ctx context.Context, id string, startTime time.Time) error
	// EndExecution updates execution end time
	EndExecution(ctx context.Context, id string, endTime time.Time, duration time.Duration) error
}
