package testresult

import (
	"context"
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
	Statuses() testkube.TestSuiteExecutionStatuses
	StatusesDefined() bool
	Page() int
	PageSize() int
	TextSearchDefined() bool
	TextSearch() string
	Selector() string
}

//go:generate mockgen -destination=./mock_repository.go -package=testresult "github.com/kubeshop/testkube/pkg/repository/testresult" Repository
type Repository interface {
	// Get gets execution result by id or name
	Get(ctx context.Context, id string) (testkube.TestSuiteExecution, error)
	// GetByNameAndTestSuite gets execution result by name
	GetByNameAndTestSuite(ctx context.Context, name, testSuiteName string) (testkube.TestSuiteExecution, error)
	// GetLatestByTestSuite gets latest execution result by test suite
	GetLatestByTestSuite(ctx context.Context, testSuiteName string) (*testkube.TestSuiteExecution, error)
	// GetLatestByTestSuites gets latest execution results by test suite names
	GetLatestByTestSuites(ctx context.Context, testSuiteNames []string) (executions []testkube.TestSuiteExecution, err error)
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
	EndExecution(ctx context.Context, execution testkube.TestSuiteExecution) error
	// DeleteByTestSuite deletes execution results by test suite
	DeleteByTestSuite(ctx context.Context, testSuiteName string) error
	// DeleteAll deletes all execution results
	DeleteAll(ctx context.Context) error
	// DeleteByTestSuites deletes execution results by test suites
	DeleteByTestSuites(ctx context.Context, testSuiteNames []string) (err error)
	// GetTestSuiteMetrics returns metrics for test suite
	GetTestSuiteMetrics(ctx context.Context, name string, limit, last int) (metrics testkube.ExecutionsMetrics, err error)
	// Count returns executions count
	Count(ctx context.Context, filter Filter) (int64, error)
}
