package v1

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/repository/result"

	"github.com/gofiber/fiber/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	executorclient "github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/log"
	logclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/server"
)

func TestTestkubeAPI_ExecutionLogsHandler(t *testing.T) {
	app := fiber.New()
	resultRepo := MockExecutionResultsRepository{}
	executor := &MockExecutor{}
	s := &TestkubeAPI{
		HTTPServer: server.HTTPServer{
			Mux: app,
			Log: log.DefaultLogger,
		},
		ExecutionResults: &resultRepo,
		ExecutorsClient:  getMockExecutorClient(),
		Executor:         executor,
	}
	app.Get("/executions/:executionID/logs", s.ExecutionLogsHandler())

	tests := []struct {
		name         string
		route        string
		expectedCode int
		execution    testkube.Execution
		jobLogs      testkube.ExecutorOutput
		wantLogs     string
	}{
		{
			name:         "Test getting execution from result output",
			route:        "/executions/finished-1234/logs",
			expectedCode: 200,
			execution: testkube.Execution{
				Id: "finished-1234",
				ExecutionResult: &testkube.ExecutionResult{
					Status: testkube.StatusPtr(testkube.PASSED_ExecutionStatus),
					Output: "storage logs",
				},
			},
			wantLogs: "storage logs",
		},
		{
			name:         "Test getting execution from job",
			route:        "/executions/running-1234/logs",
			expectedCode: 200,
			execution: testkube.Execution{
				Id:       "running-1234",
				TestType: "curl/test",
				ExecutionResult: &testkube.ExecutionResult{
					Status: testkube.StatusPtr(testkube.RUNNING_ExecutionStatus),
				},
			},
			jobLogs: testkube.ExecutorOutput{
				Type_:   output.TypeLogLine,
				Content: "job logs",
			},
			wantLogs: "job logs",
		},
	}
	responsePrefix := "data: "
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultRepo.GetFn = func(ctx context.Context, id string) (testkube.Execution, error) {
				assert.Equal(t, tt.execution.Id, id)

				return tt.execution, nil
			}
			executor.LogsFn = func(id string) (out chan output.Output, err error) {
				assert.Equal(t, tt.execution.Id, id)

				out = make(chan output.Output)
				go func() {
					defer func() {
						close(out)
					}()

					out <- output.Output(tt.jobLogs)
				}()
				return
			}

			req := httptest.NewRequest("GET", tt.route, nil)
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode, tt.name)

			b := make([]byte, len(responsePrefix))
			resp.Body.Read(b)
			assert.Equal(t, responsePrefix, string(b))

			var res output.Output
			err = json.NewDecoder(resp.Body).Decode(&res)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantLogs, res.Content)
		})
	}
}

func TestTestkubeAPI_ExecutionLogsHandlerV2(t *testing.T) {
	app := fiber.New()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	grpcClient := logclient.NewMockStreamGetter(mockCtrl)

	eventLog := events.Log{
		Content: "storage logs",
		Source:  events.SourceJobPod,
		Version: string(events.LogVersionV2),
	}

	out := make(chan events.LogResponse)
	go func() {
		defer func() {
			close(out)
		}()

		out <- events.LogResponse{Log: eventLog}
	}()

	grpcClient.EXPECT().Get(gomock.Any(), "test-execution-1").Return(out, nil)
	s := &TestkubeAPI{
		HTTPServer: server.HTTPServer{
			Mux: app,
			Log: log.DefaultLogger,
		},
		featureFlags:  featureflags.FeatureFlags{LogsV2: true},
		logGrpcClient: grpcClient,
	}
	app.Get("/executions/:executionID/logs/v2", s.ExecutionLogsHandlerV2())

	tests := []struct {
		name         string
		route        string
		expectedCode int
		eventLog     events.Log
	}{
		{
			name:         "Test getting logs from grpc client",
			route:        "/executions/test-execution-1/logs/v2",
			expectedCode: 200,
			eventLog:     eventLog,
		},
	}

	responsePrefix := "data: "
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.route, nil)
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode, tt.name)

			b := make([]byte, len(responsePrefix))
			resp.Body.Read(b)
			assert.Equal(t, responsePrefix, string(b))

			var res events.Log
			err = json.NewDecoder(resp.Body).Decode(&res)
			assert.NoError(t, err)
			assert.Equal(t, tt.eventLog, res)
		})
	}
}

type MockExecutionResultsRepository struct {
	GetFn func(ctx context.Context, id string) (testkube.Execution, error)
}

func (r MockExecutionResultsRepository) Get(ctx context.Context, id string) (testkube.Execution, error) {
	if r.GetFn == nil {
		panic("not implemented")
	}
	return r.GetFn(ctx, id)
}

func (r MockExecutionResultsRepository) GetExecution(ctx context.Context, id string) (testkube.Execution, error) {
	if r.GetFn == nil {
		panic("not implemented")
	}
	return r.GetFn(ctx, id)
}

func (r MockExecutionResultsRepository) GetByNameAndTest(ctx context.Context, name, testName string) (testkube.Execution, error) {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) GetLatestByTest(ctx context.Context, testName string) (*testkube.Execution, error) {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) GetLatestByTests(ctx context.Context, testNames []string) (executions []testkube.Execution, err error) {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) GetExecutions(ctx context.Context, filter result.Filter) ([]testkube.Execution, error) {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) GetExecutionTotals(ctx context.Context, paging bool, filter ...result.Filter) (result testkube.ExecutionsTotals, err error) {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) GetNextExecutionNumber(ctx context.Context, testName string) (int32, error) {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) Insert(ctx context.Context, result testkube.Execution) error {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) Update(ctx context.Context, result testkube.Execution) error {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) UpdateResult(ctx context.Context, id string, execution testkube.Execution) error {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) EndExecution(ctx context.Context, execution testkube.Execution) error {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) GetLabels(ctx context.Context) (labels map[string][]string, err error) {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) DeleteByTest(ctx context.Context, testName string) error {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) error {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) DeleteAll(ctx context.Context) error {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) DeleteByTests(ctx context.Context, testNames []string) error {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) error {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) DeleteForAllTestSuites(ctx context.Context) error {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) GetTestMetrics(ctx context.Context, name string, limit, last int) (testkube.ExecutionsMetrics, error) {
	panic("not implemented")
}

func (r MockExecutionResultsRepository) Count(ctx context.Context, filter result.Filter) (int64, error) {
	panic("not implemented")
}

type MockExecutor struct {
	LogsFn func(id string) (chan output.Output, error)
}

func (e MockExecutor) Execute(ctx context.Context, execution *testkube.Execution, options executorclient.ExecuteOptions) (*testkube.ExecutionResult, error) {
	panic("not implemented")
}

func (e MockExecutor) Abort(ctx context.Context, execution *testkube.Execution) (*testkube.ExecutionResult, error) {
	panic("not implemented")
}

func (e MockExecutor) Logs(ctx context.Context, id, namespace string) (chan output.Output, error) {
	if e.LogsFn == nil {
		panic("not implemented")
	}
	return e.LogsFn(id)
}

func getMockExecutorClient() *executorsclientv1.ExecutorsClient {
	scheme := runtime.NewScheme()
	executorv1.AddToScheme(scheme)

	initObjects := []k8sclient.Object{
		&executorv1.Executor{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Executor",
				APIVersion: "executor.testkube.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample",
				Namespace: "default",
			},
			Spec: executorv1.ExecutorSpec{
				Types:                []string{"curl/test"},
				ExecutorType:         "",
				JobTemplate:          "",
				JobTemplateReference: "",
			},
			Status: executorv1.ExecutorStatus{},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(initObjects...).
		Build()
	return executorsclientv1.NewClient(fakeClient, "")
}
