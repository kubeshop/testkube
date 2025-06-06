package testresult

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
)

var _ testresult.Repository = (*PostgresRepository)(nil)

const (
	PageDefaultLimit = 100
)

type PostgresRepository struct {
	db *pgxpool.Pool
	// queries *sqlc.Queries // This would be added when implementing
}

type PostgresRepositoryOpt func(*PostgresRepository)

func NewPostgresRepository(db *pgxpool.Pool, opts ...PostgresRepositoryOpt) *PostgresRepository {
	r := &PostgresRepository{
		db: db,
		// queries: sqlc.New(db), // This would be added when implementing
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Sequences interface implementation

// GetNextExecutionNumber gets next execution number by name
func (r *PostgresRepository) GetNextExecutionNumber(ctx context.Context, name string) (int32, error) {
	return 0, errors.New("GetNextExecutionNumber not implemented")
}

// Repository interface implementation

// Get gets execution result by id or name
func (r *PostgresRepository) Get(ctx context.Context, id string) (testkube.TestSuiteExecution, error) {
	return testkube.TestSuiteExecution{}, errors.New("Get not implemented")
}

// GetByNameAndTestSuite gets execution result by name
func (r *PostgresRepository) GetByNameAndTestSuite(ctx context.Context, name, testSuiteName string) (testkube.TestSuiteExecution, error) {
	return testkube.TestSuiteExecution{}, errors.New("GetByNameAndTestSuite not implemented")
}

// GetLatestByTestSuite gets latest execution result by test suite
func (r *PostgresRepository) GetLatestByTestSuite(ctx context.Context, testSuiteName string) (*testkube.TestSuiteExecution, error) {
	return nil, errors.New("GetLatestByTestSuite not implemented")
}

// GetLatestByTestSuites gets latest execution results by test suite names
func (r *PostgresRepository) GetLatestByTestSuites(ctx context.Context, testSuiteNames []string) ([]testkube.TestSuiteExecution, error) {
	return nil, errors.New("GetLatestByTestSuites not implemented")
}

// GetExecutionsTotals gets executions total stats using a filter, use filter with no data for all
func (r *PostgresRepository) GetExecutionsTotals(ctx context.Context, filter ...testresult.Filter) (testkube.ExecutionsTotals, error) {
	return testkube.ExecutionsTotals{}, errors.New("GetExecutionsTotals not implemented")
}

// GetExecutions gets executions using a filter, use filter with no data for all
func (r *PostgresRepository) GetExecutions(ctx context.Context, filter testresult.Filter) ([]testkube.TestSuiteExecution, error) {
	return nil, errors.New("GetExecutions not implemented")
}

// GetPreviousFinishedState gets previous finished execution state by test
func (r *PostgresRepository) GetPreviousFinishedState(ctx context.Context, testName string, date time.Time) (testkube.TestSuiteExecutionStatus, error) {
	return "", errors.New("GetPreviousFinishedState not implemented")
}

// Insert inserts new execution result
func (r *PostgresRepository) Insert(ctx context.Context, result testkube.TestSuiteExecution) error {
	return errors.New("Insert not implemented")
}

// Update updates execution result
func (r *PostgresRepository) Update(ctx context.Context, result testkube.TestSuiteExecution) error {
	return errors.New("Update not implemented")
}

// StartExecution updates execution start time
func (r *PostgresRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
	return errors.New("StartExecution not implemented")
}

// EndExecution updates execution end time
func (r *PostgresRepository) EndExecution(ctx context.Context, execution testkube.TestSuiteExecution) error {
	return errors.New("EndExecution not implemented")
}

// DeleteByTestSuite deletes execution results by test suite
func (r *PostgresRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) error {
	return errors.New("DeleteByTestSuite not implemented")
}

// DeleteAll deletes all execution results
func (r *PostgresRepository) DeleteAll(ctx context.Context) error {
	return errors.New("DeleteAll not implemented")
}

// DeleteByTestSuites deletes execution results by test suites
func (r *PostgresRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) error {
	return errors.New("DeleteByTestSuites not implemented")
}

// GetTestSuiteMetrics returns metrics for test suite
func (r *PostgresRepository) GetTestSuiteMetrics(ctx context.Context, name string, limit, last int) (testkube.ExecutionsMetrics, error) {
	return testkube.ExecutionsMetrics{}, errors.New("GetTestSuiteMetrics not implemented")
}

// Count returns executions count
func (r *PostgresRepository) Count(ctx context.Context, filter testresult.Filter) (int64, error) {
	return 0, errors.New("Count not implemented")
}
