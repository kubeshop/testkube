package containerexecutor

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	templatesclientv1 "github.com/kubeshop/testkube-operator/pkg/client/templates/v1"
	v3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	"github.com/kubeshop/testkube/internal/featureflags"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/executor/options"
	"github.com/kubeshop/testkube/pkg/repository/result"
)

var ctx = context.Background()

func TestExecuteAsync(t *testing.T) {
	t.Parallel()

	ce := ContainerExecutor{
		clientSet:       getFakeClient("1"),
		log:             logger(),
		repository:      FakeResultRepository{},
		metrics:         FakeMetricCounter{},
		emitter:         FakeEmitter{},
		namespace:       "default",
		configMap:       FakeConfigRepository{},
		testsClient:     FakeTestsClient{},
		executorsClient: FakeExecutorsClient{},
	}

	execution := &testkube.Execution{Id: "1"}
	opts := options.ExecuteOptions{}
	res, err := ce.Execute(ctx, execution, opts)
	assert.NoError(t, err)

	// Status is either running or passed, depends if async goroutine managed to finish
	assert.Contains(t,
		[]testkube.ExecutionStatus{testkube.RUNNING_ExecutionStatus, testkube.PASSED_ExecutionStatus},
		*res.Status)
}

func TestExecuteSync(t *testing.T) {
	t.Parallel()

	ce := ContainerExecutor{
		clientSet:       getFakeClient("1"),
		log:             logger(),
		repository:      FakeResultRepository{},
		metrics:         FakeMetricCounter{},
		emitter:         FakeEmitter{},
		namespace:       "default",
		configMap:       FakeConfigRepository{},
		testsClient:     FakeTestsClient{},
		executorsClient: FakeExecutorsClient{},
	}

	execution := &testkube.Execution{Id: "1"}
	opts := options.ExecuteOptions{
		ImagePullSecretNames: []string{"secret-name1"},
		Sync:                 true,
	}
	res, err := ce.Execute(ctx, execution, opts)
	assert.NoError(t, err)
	assert.Equal(t, testkube.PASSED_ExecutionStatus, *res.Status)
}

func logger() *zap.SugaredLogger {
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zap.DebugLevel)

	zapCfg := zap.NewDevelopmentConfig()
	zapCfg.Level = atomicLevel

	z, err := zapCfg.Build()
	if err != nil {
		panic(err)
	}
	return z.Sugar()
}

func getFakeClient(executionID string) *fake.Clientset {
	initObjects := []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      executionID,
				Namespace: "default",
				Labels: map[string]string{
					"job-name": executionID,
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodSucceeded,
			},
		},
	}
	fakeClient := fake.NewSimpleClientset(initObjects...)
	return fakeClient
}

type FakeMetricCounter struct {
}

func (FakeMetricCounter) IncExecuteTest(execution testkube.Execution, dashboardURI string) {
}

type FakeEmitter struct {
}

func (FakeEmitter) Notify(event testkube.Event) {
}

type FakeResultRepository struct {
}

func (r FakeResultRepository) GetNextExecutionNumber(ctx context.Context, testName string) (number int32, err error) {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) GetByNameAndTest(ctx context.Context, name, testName string) (testkube.Execution, error) {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) GetLatestByTest(ctx context.Context, testName string) (*testkube.Execution, error) {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) GetLatestByTests(ctx context.Context, testNames []string) (executions []testkube.Execution, err error) {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) GetExecutions(ctx context.Context, filter result.Filter) ([]testkube.Execution, error) {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) GetExecutionTotals(ctx context.Context, paging bool, filter ...result.Filter) (result testkube.ExecutionsTotals, err error) {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) Insert(ctx context.Context, result testkube.Execution) error {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) Update(ctx context.Context, result testkube.Execution) error {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) GetLabels(ctx context.Context) (labels map[string][]string, err error) {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) DeleteByTest(ctx context.Context, testName string) error {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) DeleteByTestSuite(ctx context.Context, testSuiteName string) error {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) DeleteAll(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) DeleteByTests(ctx context.Context, testNames []string) (err error) {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) DeleteByTestSuites(ctx context.Context, testSuiteNames []string) (err error) {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) DeleteForAllTestSuites(ctx context.Context) (err error) {
	//TODO implement me
	panic("implement me")
}

func (r FakeResultRepository) GetTestMetrics(ctx context.Context, name string, limit, last int) (metrics testkube.ExecutionsMetrics, err error) {
	//TODO implement me
	panic("implement me")
}

func (FakeResultRepository) Get(ctx context.Context, id string) (testkube.Execution, error) {
	return testkube.Execution{}, nil
}

func (FakeResultRepository) UpdateResult(ctx context.Context, id string, execution testkube.Execution) error {
	return nil
}
func (FakeResultRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
	return nil
}
func (FakeResultRepository) EndExecution(ctx context.Context, execution testkube.Execution) error {
	return nil
}

type FakeConfigRepository struct {
}

func (FakeConfigRepository) GetUniqueClusterId(ctx context.Context) (string, error) {
	return "", nil
}

func (FakeConfigRepository) GetTelemetryEnabled(ctx context.Context) (ok bool, err error) {
	return false, nil
}

func (FakeConfigRepository) Get(ctx context.Context) (testkube.Config, error) {
	return testkube.Config{}, nil
}

func (FakeConfigRepository) Upsert(ctx context.Context, config testkube.Config) (testkube.Config, error) {
	return config, nil
}

type FakeTestsClient struct {
}

func (FakeTestsClient) List(selector string) (*testsv3.TestList, error) {
	return &testsv3.TestList{}, nil
}

func (FakeTestsClient) ListLabels() (map[string][]string, error) {
	return map[string][]string{}, nil
}

func (FakeTestsClient) Get(name string) (*testsv3.Test, error) {
	return &testsv3.Test{}, nil
}

func (FakeTestsClient) Create(test *testsv3.Test, options ...v3.Option) (*testsv3.Test, error) {
	return &testsv3.Test{}, nil
}

func (FakeTestsClient) Update(test *testsv3.Test, options ...v3.Option) (*testsv3.Test, error) {
	return &testsv3.Test{}, nil
}

func (FakeTestsClient) Delete(name string) error {
	return nil
}

func (FakeTestsClient) DeleteAll() error {
	return nil
}

func (FakeTestsClient) CreateTestSecrets(test *testsv3.Test) error {
	return nil
}

func (FakeTestsClient) UpdateTestSecrets(test *testsv3.Test) error {
	return nil
}

func (FakeTestsClient) LoadTestVariablesSecret(test *testsv3.Test) (*corev1.Secret, error) {
	return &corev1.Secret{}, nil
}

func (FakeTestsClient) GetCurrentSecretUUID(testName string) (string, error) {
	return "", nil
}

func (FakeTestsClient) GetSecretTestVars(testName, secretUUID string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (FakeTestsClient) ListByNames(names []string) ([]testsv3.Test, error) {
	return []testsv3.Test{}, nil
}

func (FakeTestsClient) DeleteByLabels(selector string) error {
	return nil
}

func (FakeTestsClient) UpdateStatus(test *testsv3.Test) error {
	return nil
}

type FakeExecutorsClient struct {
}

func (FakeExecutorsClient) List(selector string) (*executorv1.ExecutorList, error) {
	return &executorv1.ExecutorList{}, nil
}

func (FakeExecutorsClient) Get(name string) (*executorv1.Executor, error) {
	return &executorv1.Executor{}, nil
}

func (FakeExecutorsClient) GetByType(executorType string) (*executorv1.Executor, error) {
	return &executorv1.Executor{}, nil
}

func (FakeExecutorsClient) Create(executor *executorv1.Executor) (*executorv1.Executor, error) {
	return &executorv1.Executor{}, nil
}

func (FakeExecutorsClient) Delete(name string) error {
	return nil
}

func (FakeExecutorsClient) Update(executor *executorv1.Executor) (*executorv1.Executor, error) {
	return &executorv1.Executor{}, nil
}

func (FakeExecutorsClient) DeleteByLabels(selector string) error {
	return nil
}
