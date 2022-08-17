package v1

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	testsuitesv2 "github.com/kubeshop/testkube-operator/apis/testsuite/v2"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/testresult"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/server"
	"github.com/stretchr/testify/assert"
)

func TestTestkubeAPI_GetTestSuiteWithExecutionHandler(t *testing.T) {
	app := fiber.New()
	route := "/test-suite-with-executions"
	resultRepo := MockTestExecutionRepository{}
	testSuitesClient := &MockTestSuitesClient{}
	s := &TestkubeAPI{
		HTTPServer: server.HTTPServer{
			Mux: app,
			Log: log.DefaultLogger,
		},
		TestExecutionResults: &resultRepo,
		TestsSuitesClient:    testSuitesClient,
	}
	app.Get(route, s.ListTestSuiteWithExecutionsHandler())

	tests := []struct {
		name         string
		expectedCode int
		list         []testkube.TestSuiteWithExecution
		want         []testkube.TestSuiteWithExecution
	}{
		{
			name:         "No test suites should return empty list",
			expectedCode: 200,
			list:         []testkube.TestSuiteWithExecution{},
			want:         []testkube.TestSuiteWithExecution{},
		},
		{
			name:         "No execution on test suite should return list with no executions",
			expectedCode: 200,
			list:         []testkube.TestSuiteWithExecution{},
			want:         []testkube.TestSuiteWithExecution{},
		},
		{
			name:         "Test suites with executions should be returned fully",
			expectedCode: 200,
			list:         []testkube.TestSuiteWithExecution{},
			want:         []testkube.TestSuiteWithExecution{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSuitesClient.ListFn = func(selector string) (*testsuitesv2.TestSuiteList, error) {
				return &testsuitesv2.TestSuiteList{}, nil
			}
			resultRepo.GetLatestByTestSuitesFn = func(ctx context.Context, testSuiteNames []string, sortField string) (executions []testkube.TestSuiteExecution, err error) {
				return []testkube.TestSuiteExecution{}, nil
			}

			req := httptest.NewRequest("GET", route, nil)
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode, tt.name)

			var res []testkube.TestSuiteWithExecution
			err = json.NewDecoder(resp.Body).Decode(&res)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

type MockTestExecutionRepository struct {
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
	GetTestSuiteMetricsFn   func(ctx context.Context, name string, limit int) (metrics testkube.TestSuiteMetrics, err error)
}

func (r MockTestExecutionRepository) Get(ctx context.Context, id string) (testkube.TestSuiteExecution, error) {
	if r.GetFn == nil {
		panic("not implemented")
	}
	return r.GetFn(ctx, id)
}

func (r MockTestExecutionRepository) GetByNameAndTestSuite(ctx context.Context, name, testSuiteName string) (testkube.TestSuiteExecution, error) {
	if r.GetByNameAndTestSuiteFn == nil {
		panic("not implemented")
	}
	return r.GetByNameAndTestSuiteFn(ctx, name, testSuiteName)
}

func (r MockTestExecutionRepository) GetLatestByTestSuite(ctx context.Context, testSuiteName, sortField string) (testkube.TestSuiteExecution, error) {
	if r.GetLatestByTestSuiteFn == nil {
		panic("not implemented")
	}
	return r.GetLatestByTestSuiteFn(ctx, testSuiteName, sortField)
}

func (r MockTestExecutionRepository) GetLatestByTestSuites(ctx context.Context, testSuiteNames []string, sortField string) (executions []testkube.TestSuiteExecution, err error) {
	if r.GetLatestByTestSuitesFn == nil {
		panic("not implemented")
	}
	return r.GetLatestByTestSuitesFn(ctx, testSuiteNames, sortField)
}

func (r MockTestExecutionRepository) GetExecutionsTotals(ctx context.Context, filter ...testresult.Filter) (totals testkube.ExecutionsTotals, err error) {
	if r.GetExecutionsTotalsFn == nil {
		panic("not implemented")
	}
	return r.GetExecutionsTotalsFn(ctx, filter...)
}

func (r MockTestExecutionRepository) GetExecutions(ctx context.Context, filter testresult.Filter) ([]testkube.TestSuiteExecution, error) {
	if r.GetExecutionsFn == nil {
		panic("not implemented")
	}
	return r.GetExecutionsFn(ctx, filter)
}

func (r MockTestExecutionRepository) Insert(ctx context.Context, result testkube.TestSuiteExecution) error {
	if r.InsertFn == nil {
		panic("not implemented")
	}
	return r.InsertFn(ctx, result)
}

func (r MockTestExecutionRepository) Update(ctx context.Context, result testkube.TestSuiteExecution) error {
	if r.UpdateFn == nil {
		panic("not implemented")
	}
	return r.UpdateFn(ctx, result)
}

func (r MockTestExecutionRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
	if r.StartExecutionFn == nil {
		panic("not implemented")
	}
	return r.StartExecutionFn(ctx, id, startTime)
}

func (r MockTestExecutionRepository) EndExecution(ctx context.Context, id string, endTime time.Time, duration time.Duration) error {
	if r.EndExecutionFn == nil {
		panic("not implemented")
	}
	return r.EndExecutionFn(ctx, id, endTime, duration)
}

func (r MockTestExecutionRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) error {
	if r.DeleteByTestSuiteFn == nil {
		panic("not implemented")
	}
	return r.DeleteByTestSuiteFn(ctx, testSuiteName)
}

func (r MockTestExecutionRepository) DeleteAll(ctx context.Context) error {
	if r.DeleteAllFn == nil {
		panic("not implemented")
	}
	return r.DeleteAllFn(ctx)
}

func (r MockTestExecutionRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) (err error) {
	if r.DeleteByTestSuitesFn == nil {
		panic("not implemented")
	}
	return r.DeleteByTestSuitesFn(ctx, testSuiteNames)
}

func (r MockTestExecutionRepository) GetTestSuiteMetrics(ctx context.Context, name string, limit int) (metrics testkube.TestSuiteMetrics, err error) {
	if r.GetTestSuiteMetricsFn == nil {
		panic("not implemented")
	}
	return r.GetTestSuiteMetricsFn(ctx, name, limit)
}

type MockTestSuitesClient struct {
	CreateFn                 func(testsuite *testsuitesv2.TestSuite) (*testsuitesv2.TestSuite, error)
	GetFn                    func(name string) (*testsuitesv2.TestSuite, error)
	GetSecretTestSuiteVarsFn func(testSuiteName, secretUUID string) (map[string]string, error)
	GetCurrentSecretUUIDFn   func(testSuiteName string) (string, error)
	ListFn                   func(selector string) (*testsuitesv2.TestSuiteList, error)
	ListLabelsFn             func() (map[string][]string, error)
	UpdateFn                 func(testsuite *testsuitesv2.TestSuite) (*testsuitesv2.TestSuite, error)
	DeleteFn                 func(name string) error
	DeleteAllFn              func() error
	DeleteByLabelsFn         func(selector string) error
}

func (c MockTestSuitesClient) Create(testsuite *testsuitesv2.TestSuite) (*testsuitesv2.TestSuite, error) {
	if c.CreateFn == nil {
		panic("not implemented")
	}
	return c.CreateFn(testsuite)
}

func (c MockTestSuitesClient) Get(name string) (*testsuitesv2.TestSuite, error) {
	if c.GetFn == nil {
		panic("not implemented")
	}
	return c.GetFn(name)
}
func (c MockTestSuitesClient) GetSecretTestSuiteVars(testSuiteName, secretUUID string) (map[string]string, error) {
	if c.GetSecretTestSuiteVarsFn == nil {
		panic("not implemented")
	}
	return c.GetSecretTestSuiteVarsFn(testSuiteName, secretUUID)
}

func (c MockTestSuitesClient) GetCurrentSecretUUID(testSuiteName string) (string, error) {
	if c.GetCurrentSecretUUIDFn == nil {
		panic("not implemented")
	}
	return c.GetCurrentSecretUUIDFn(testSuiteName)
}

func (c MockTestSuitesClient) List(selector string) (*testsuitesv2.TestSuiteList, error) {
	if c.ListFn == nil {
		panic("not implemented")
	}
	return c.ListFn(selector)
}

func (c MockTestSuitesClient) ListLabels() (map[string][]string, error) {
	if c.ListLabelsFn == nil {
		panic("not implemented")
	}
	return c.ListLabelsFn()
}

func (c MockTestSuitesClient) Update(testsuite *testsuitesv2.TestSuite) (*testsuitesv2.TestSuite, error) {
	if c.UpdateFn == nil {
		panic("not implemented")
	}
	return c.UpdateFn(testsuite)
}

func (c MockTestSuitesClient) Delete(name string) error {
	if c.DeleteFn == nil {
		panic("not implemented")
	}
	return c.DeleteFn(name)
}

func (c MockTestSuitesClient) DeleteAll() error {
	if c.DeleteAllFn == nil {
		panic("not implemented")
	}
	return c.DeleteAllFn()
}

func (c MockTestSuitesClient) DeleteByLabels(selector string) error {
	if c.DeleteByLabelsFn == nil {
		panic("not implemented")
	}
	return c.DeleteByLabelsFn(selector)
}
