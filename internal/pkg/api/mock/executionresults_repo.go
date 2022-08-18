package mock

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type ExecutionResultsRepository struct {
	GetFn func(ctx context.Context, id string) (testkube.Execution, error)
}

func (r ExecutionResultsRepository) Get(ctx context.Context, id string) (testkube.Execution, error) {
	if r.GetFn == nil {
		panic("not implemented")
	}
	return r.GetFn(ctx, id)
}

func (r ExecutionResultsRepository) GetByName(ctx context.Context, name string) (testkube.Execution, error) {
	if r.GetFn == nil {
		panic("not implemented")
	}
	return r.GetFn(ctx, name)
}

func (r ExecutionResultsRepository) GetByNameAndTest(ctx context.Context, name, testName string) (testkube.Execution, error) {
	panic("not implemented")
}

func (r ExecutionResultsRepository) GetLatestByTest(ctx context.Context, testName, sortField string) (testkube.Execution, error) {
	panic("not implemented")
}

func (r ExecutionResultsRepository) GetLatestByTests(ctx context.Context, testNames []string, sortField string) (executions []testkube.Execution, err error) {
	panic("not implemented")
}

func (r ExecutionResultsRepository) GetExecutions(ctx context.Context, filter result.Filter) ([]testkube.Execution, error) {
	panic("not implemented")
}

func (r ExecutionResultsRepository) GetExecutionTotals(ctx context.Context, paging bool, filter ...result.Filter) (result testkube.ExecutionsTotals, err error) {
	panic("not implemented")
}

func (r ExecutionResultsRepository) GetNextExecutionNumber(ctx context.Context, testName string) (int32, error) {
	panic("not implemented")
}

func (r ExecutionResultsRepository) Insert(ctx context.Context, result testkube.Execution) error {
	panic("not implemented")
}

func (r ExecutionResultsRepository) Update(ctx context.Context, result testkube.Execution) error {
	panic("not implemented")
}

func (r ExecutionResultsRepository) UpdateResult(ctx context.Context, id string, execution testkube.ExecutionResult) error {
	panic("not implemented")
}

func (r ExecutionResultsRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
	panic("not implemented")
}

func (r ExecutionResultsRepository) EndExecution(ctx context.Context, id string, endTime time.Time, duration time.Duration) error {
	panic("not implemented")
}

func (r ExecutionResultsRepository) GetLabels(ctx context.Context) (labels map[string][]string, err error) {
	panic("not implemented")
}

func (r ExecutionResultsRepository) DeleteByTest(ctx context.Context, testName string) error {
	panic("not implemented")
}

func (r ExecutionResultsRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) error {
	panic("not implemented")
}

func (r ExecutionResultsRepository) DeleteAll(ctx context.Context) error {
	panic("not implemented")
}

func (r ExecutionResultsRepository) DeleteByTests(ctx context.Context, testNames []string) error {
	panic("not implemented")
}

func (r ExecutionResultsRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) error {
	panic("not implemented")
}

func (r ExecutionResultsRepository) DeleteForAllTestSuites(ctx context.Context) error {
	panic("not implemented")
}

func (r ExecutionResultsRepository) GetTestMetrics(ctx context.Context, name string, limit int) (testkube.ExecutionsMetrics, error) {
	panic("not implemented")
}
