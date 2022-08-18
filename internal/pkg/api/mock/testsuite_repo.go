package mock

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/internal/pkg/api/repository/testresult"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type TestSuiteRepository struct {
	GetFn                   func(ctx context.Context, id string) (testkube.TestSuiteExecution, error)
	GetByNameAndTestSuiteFn func(ctx context.Context, name, testSuiteName string) (testkube.TestSuiteExecution, error)
	GetLatestByTestSuiteFn  func(ctx context.Context, testSuiteName, sortField string) (testkube.TestSuiteExecution, error)
	GetLatestByTestSuitesFn func(ctx context.Context, testSuiteNames []string, sortField string) (executions []testkube.TestSuiteExecution, err error)
	GetExecutionsTotalsFn   func(ctx context.Context, filter ...testresult.Filter) (totals testkube.ExecutionsTotals, err error)
	GetExecutionsFn         func(ctx context.Context, filter testresult.Filter) ([]testkube.TestSuiteExecution, error)
	InsertFn                func(ctx context.Context, result testkube.TestSuiteExecution) error
	UpdateFn                func(ctx context.Context, result testkube.TestSuiteExecution) error
	StartExecutionFn        func(ctx context.Context, id string, startTime time.Time) error
	EndExecutionFn          func(ctx context.Context, id string, endTime time.Time, duration time.Duration) error
	DeleteByTestSuiteFn     func(ctx context.Context, testSuiteName string) error
	DeleteAllFn             func(ctx context.Context) error
	DeleteByTestSuitesFn    func(ctx context.Context, testSuiteNames []string) (err error)
	GetTestSuiteMetricsFn   func(ctx context.Context, name string, limit int) (metrics testkube.ExecutionsMetrics, err error)
}

func (r TestSuiteRepository) Get(ctx context.Context, id string) (testkube.TestSuiteExecution, error) {
	if r.GetFn == nil {
		panic("not implemented")
	}
	return r.GetFn(ctx, id)
}

func (r TestSuiteRepository) GetByNameAndTestSuite(ctx context.Context, name, testSuiteName string) (testkube.TestSuiteExecution, error) {
	if r.GetByNameAndTestSuiteFn == nil {
		panic("not implemented")
	}
	return r.GetByNameAndTestSuiteFn(ctx, name, testSuiteName)
}

func (r TestSuiteRepository) GetLatestByTestSuite(ctx context.Context, testSuiteName, sortField string) (testkube.TestSuiteExecution, error) {
	if r.GetLatestByTestSuiteFn == nil {
		panic("not implemented")
	}
	return r.GetLatestByTestSuiteFn(ctx, testSuiteName, sortField)
}

func (r TestSuiteRepository) GetLatestByTestSuites(ctx context.Context, testSuiteNames []string, sortField string) (executions []testkube.TestSuiteExecution, err error) {
	if r.GetLatestByTestSuitesFn == nil {
		panic("not implemented")
	}
	return r.GetLatestByTestSuitesFn(ctx, testSuiteNames, sortField)
}

func (r TestSuiteRepository) GetExecutionsTotals(ctx context.Context, filter ...testresult.Filter) (totals testkube.ExecutionsTotals, err error) {
	if r.GetExecutionsTotalsFn == nil {
		panic("not implemented")
	}
	return r.GetExecutionsTotalsFn(ctx, filter...)
}

func (r TestSuiteRepository) GetExecutions(ctx context.Context, filter testresult.Filter) ([]testkube.TestSuiteExecution, error) {
	if r.GetExecutionsFn == nil {
		panic("not implemented")
	}
	return r.GetExecutionsFn(ctx, filter)
}

func (r TestSuiteRepository) Insert(ctx context.Context, result testkube.TestSuiteExecution) error {
	if r.InsertFn == nil {
		panic("not implemented")
	}
	return r.InsertFn(ctx, result)
}

func (r TestSuiteRepository) Update(ctx context.Context, result testkube.TestSuiteExecution) error {
	if r.UpdateFn == nil {
		panic("not implemented")
	}
	return r.UpdateFn(ctx, result)
}

func (r TestSuiteRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
	if r.StartExecutionFn == nil {
		panic("not implemented")
	}
	return r.StartExecutionFn(ctx, id, startTime)
}

func (r TestSuiteRepository) EndExecution(ctx context.Context, id string, endTime time.Time, duration time.Duration) error {
	if r.EndExecutionFn == nil {
		panic("not implemented")
	}
	return r.EndExecutionFn(ctx, id, endTime, duration)
}

func (r TestSuiteRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) error {
	if r.DeleteByTestSuiteFn == nil {
		panic("not implemented")
	}
	return r.DeleteByTestSuiteFn(ctx, testSuiteName)
}

func (r TestSuiteRepository) DeleteAll(ctx context.Context) error {
	if r.DeleteAllFn == nil {
		panic("not implemented")
	}
	return r.DeleteAllFn(ctx)
}

func (r TestSuiteRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) (err error) {
	if r.DeleteByTestSuitesFn == nil {
		panic("not implemented")
	}
	return r.DeleteByTestSuitesFn(ctx, testSuiteNames)
}

func (r TestSuiteRepository) GetTestSuiteMetrics(ctx context.Context, name string, limit int) (metrics testkube.ExecutionsMetrics, err error) {
	if r.GetTestSuiteMetricsFn == nil {
		panic("not implemented")
	}
	return r.GetTestSuiteMetricsFn(ctx, name, limit)
}
