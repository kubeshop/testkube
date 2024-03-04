package result

import (
	"context"
	"io"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const PageDefaultLimit int = 100

type Filter interface {
	TestName() string
	TestNameDefined() bool
	LastNDays() int
	LastNDaysDefined() bool
	StartDate() time.Time
	StartDateDefined() bool
	EndDate() time.Time
	EndDateDefined() bool
	Statuses() testkube.ExecutionStatuses
	StatusesDefined() bool
	Page() int
	PageSize() int
	TextSearchDefined() bool
	TextSearch() string
	Selector() string
	TypeDefined() bool
	Type() string
}

//go:generate mockgen -destination=./mock_repository.go -package=result "github.com/kubeshop/testkube/pkg/repository/result" Repository
type Repository interface {
	Sequences
	// Get gets execution result by id or name
	Get(ctx context.Context, id string) (testkube.Execution, error)
	// Get gets execution result without output
	GetExecution(ctx context.Context, id string) (testkube.Execution, error)
	// GetByNameAndTest gets execution result by name and test name
	GetByNameAndTest(ctx context.Context, name, testName string) (testkube.Execution, error)
	// GetLatestByTest gets latest execution result by test
	GetLatestByTest(ctx context.Context, testName string) (*testkube.Execution, error)
	// GetLatestByTests gets latest execution results by test names
	GetLatestByTests(ctx context.Context, testNames []string) (executions []testkube.Execution, err error)
	// GetExecutions gets executions using a filter, use filter with no data for all
	GetExecutions(ctx context.Context, filter Filter) ([]testkube.Execution, error)
	// GetExecutionTotals gets the statistics on number of executions using a filter, but without paging
	GetExecutionTotals(ctx context.Context, paging bool, filter ...Filter) (result testkube.ExecutionsTotals, err error)
	// Insert inserts new execution result
	Insert(ctx context.Context, result testkube.Execution) error
	// Update updates execution result
	Update(ctx context.Context, result testkube.Execution) error
	// UpdateResult updates result in execution
	UpdateResult(ctx context.Context, id string, execution testkube.Execution) error
	// StartExecution updates execution start time
	StartExecution(ctx context.Context, id string, startTime time.Time) error
	// EndExecution updates execution end time
	EndExecution(ctx context.Context, execution testkube.Execution) error
	// GetLabels get all available labels
	GetLabels(ctx context.Context) (labels map[string][]string, err error)
	// DeleteByTest deletes execution results by test
	DeleteByTest(ctx context.Context, testName string) error
	// DeleteByTestSuite deletes execution results by test suite
	DeleteByTestSuite(ctx context.Context, testSuiteName string) error
	// DeleteAll deletes all execution results
	DeleteAll(ctx context.Context) error
	// DeleteByTests deletes execution results by tests
	DeleteByTests(ctx context.Context, testNames []string) (err error)
	// DeleteByTestSuites deletes execution results by test suites
	DeleteByTestSuites(ctx context.Context, testSuiteNames []string) (err error)
	// DeleteForAllTestSuites deletes execution results for all test suites
	DeleteForAllTestSuites(ctx context.Context) (err error)
	// GetTestMetrics returns metrics for test
	GetTestMetrics(ctx context.Context, name string, limit, last int) (metrics testkube.ExecutionsMetrics, err error)
	// Count returns executions count
	Count(ctx context.Context, filter Filter) (int64, error)
}

type Sequences interface {
	// GetNextExecutionNumber gets next execution number by test name
	GetNextExecutionNumber(ctx context.Context, testName string) (number int32, err error)
}

//go:generate mockgen -destination=./mock_output_repository.go -package=result "github.com/kubeshop/testkube/pkg/repository/result" OutputRepository
type OutputRepository interface {
	// GetOutput gets execution output by id or name
	GetOutput(ctx context.Context, id, testName, testSuiteName string) (output string, err error)
	// InsertOutput inserts new execution output
	InsertOutput(ctx context.Context, id, testName, testSuiteName, output string) error
	// UpdateOutput updates execution output
	UpdateOutput(ctx context.Context, id, testName, testSuiteName, output string) error
	// DeleteOutput deletes execution output
	DeleteOutput(ctx context.Context, id, testName, testSuiteName string) error
	// DeleteOutputByTest deletes execution output by test
	DeleteOutputByTest(ctx context.Context, testName string) error
	// DeleteOutputForTests deletes execution output for tests
	DeleteOutputForTests(ctx context.Context, testNames []string) error
	// DeleteOutputByTestSuite deletes execution output by test suite
	DeleteOutputByTestSuite(ctx context.Context, testSuiteName string) error
	// DeleteOutputForTestSuites deletes execution output for test suites
	DeleteOutputForTestSuites(ctx context.Context, testSuiteNames []string) error
	// DeleteAllOutput deletes all execution output
	DeleteAllOutput(ctx context.Context) error
	// DeleteOutputForAllTestSuite deletes all execution output for test suite
	DeleteOutputForAllTestSuite(ctx context.Context) error
	// StreamOutput streams execution output by id or name
	StreamOutput(ctx context.Context, executionID, testName, testSuiteName string) (reader io.Reader, err error)
	// GetOutputSize gets execution output metadata by id or name
	GetOutputSize(ctx context.Context, executionID, testName, testSuiteName string) (size int, err error)
}
