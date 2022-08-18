package mock

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type ExecutionRepository struct {
	GetFn          func(ctx context.Context, id string) (testkube.Execution, error)
	EndExecutionFn func(ctx context.Context, id string, endTime time.Time, duration time.Duration) error
}

func (r ExecutionRepository) Get(ctx context.Context, id string) (testkube.Execution, error) {
	if r.GetFn == nil {
		panic("not implemented")
	}
	return r.GetFn(ctx, id)
}

func (r ExecutionRepository) GetByName(ctx context.Context, name string) (testkube.Execution, error) {
	if r.GetFn == nil {
		panic("not implemented")
	}
	return r.GetFn(ctx, name)
}

func (r ExecutionRepository) GetByNameAndTest(ctx context.Context, name, testName string) (testkube.Execution, error) {
	panic("not implemented")
}

func (r ExecutionRepository) GetLatestByTest(ctx context.Context, testName, sortField string) (testkube.Execution, error) {
	panic("not implemented")
}

func (r ExecutionRepository) GetLatestByTests(ctx context.Context, testNames []string, sortField string) (executions []testkube.Execution, err error) {
	panic("not implemented")
}

func (r ExecutionRepository) GetExecutions(ctx context.Context, filter result.Filter) ([]testkube.Execution, error) {
	panic("not implemented")
}

func (r ExecutionRepository) GetExecutionTotals(ctx context.Context, paging bool, filter ...result.Filter) (result testkube.ExecutionsTotals, err error) {
	panic("not implemented")
}

func (r ExecutionRepository) GetNextExecutionNumber(ctx context.Context, testName string) (int32, error) {
	panic("not implemented")
}

func (r ExecutionRepository) Insert(ctx context.Context, result testkube.Execution) error {
	panic("not implemented")
}

func (r ExecutionRepository) Update(ctx context.Context, result testkube.Execution) error {
	panic("not implemented")
}

func (r ExecutionRepository) UpdateResult(ctx context.Context, id string, execution testkube.ExecutionResult) error {
	panic("not implemented")
}

func (r ExecutionRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
	panic("not implemented")
}

func (r ExecutionRepository) EndExecution(ctx context.Context, id string, endTime time.Time, duration time.Duration) error {
	if r.EndExecutionFn == nil {
		panic("not implemented")
	}
	return r.EndExecutionFn(ctx, id, endTime, duration)
}

func (r ExecutionRepository) GetLabels(ctx context.Context) (labels map[string][]string, err error) {
	panic("not implemented")
}

func (r ExecutionRepository) DeleteByTest(ctx context.Context, testName string) error {
	panic("not implemented")
}

func (r ExecutionRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) error {
	panic("not implemented")
}

func (r ExecutionRepository) DeleteAll(ctx context.Context) error {
	panic("not implemented")
}

func (r ExecutionRepository) DeleteByTests(ctx context.Context, testNames []string) error {
	panic("not implemented")
}

func (r ExecutionRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) error {
	panic("not implemented")
}

func (r ExecutionRepository) DeleteForAllTestSuites(ctx context.Context) error {
	panic("not implemented")
}

func (r ExecutionRepository) GetTestMetrics(ctx context.Context, name string, limit int) (testkube.ExecutionsMetrics, error) {
	panic("not implemented")
}
