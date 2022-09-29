package containerexecutor

import (
	"context"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestExecuteAsync(t *testing.T) {
	ce := ContainerExecutor{
		clientSet:  getFakeClient("1"),
		log:        logger(),
		repository: FakeResultRepository{},
		metrics:    FakeMetricCounter{},
		emitter:    FakeEmitter{},
		namespace:  "default",
	}

	execution := &testkube.Execution{Id: "1"}
	options := client.ExecuteOptions{}
	res, err := ce.Execute(execution, options)
	assert.NoError(t, err)

	// Status is either running or passed, depending if async goroutine managed to finish
	assert.Contains(t,
		[]testkube.ExecutionStatus{testkube.RUNNING_ExecutionStatus, testkube.PASSED_ExecutionStatus},
		*res.Status)
}

func TestExecuteSync(t *testing.T) {
	ce := ContainerExecutor{
		clientSet:  getFakeClient("1"),
		log:        logger(),
		repository: FakeResultRepository{},
		metrics:    FakeMetricCounter{},
		emitter:    FakeEmitter{},
		namespace:  "default",
	}

	execution := &testkube.Execution{Id: "1"}
	options := client.ExecuteOptions{ImagePullSecretNames: []string{"secret-name1"}}
	res, err := ce.ExecuteSync(execution, options)
	assert.NoError(t, err)
	assert.Equal(t, testkube.PASSED_ExecutionStatus, *res.Status)
}

func TestNewJobSpecEmptyArgs(t *testing.T) {
	jobOptions := &JobOptions{
		Name:      "name",
		Namespace: "namespace",
		InitImage: "kubeshop/testkube-executor-init:0.7.10",
		Image:     "ubuntu",
		Args:      []string{},
	}
	spec, err := NewJobSpec(logger(), jobOptions)
	assert.NoError(t, err)
	assert.NotNil(t, spec)
}

func TestNewJobSpecWithArgs(t *testing.T) {
	jobOptions := &JobOptions{
		Name:             "name",
		Namespace:        "namespace",
		InitImage:        "kubeshop/testkube-executor-init:0.7.10",
		Image:            "curl",
		ImagePullSecrets: []string{"secret-name"},
		Command:          []string{"/bin/curl"},
		Args:             []string{"-v", "https://testkube.kubeshop.io"},
	}
	spec, err := NewJobSpec(logger(), jobOptions)
	assert.NoError(t, err)
	assert.NotNil(t, spec)
}

func TestNewJobSpecWithoutInitImage(t *testing.T) {
	jobOptions := &JobOptions{
		Name:      "name",
		Namespace: "namespace",
		InitImage: "",
		Image:     "ubuntu",
		Args:      []string{},
	}
	spec, err := NewJobSpec(logger(), jobOptions)
	assert.NoError(t, err)
	assert.NotNil(t, spec)
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

func (FakeMetricCounter) IncExecuteTest(execution testkube.Execution) {
	return
}

type FakeEmitter struct {
}

func (FakeEmitter) Notify(event testkube.Event) {
	return
}

type FakeResultRepository struct {
}

func (FakeResultRepository) UpdateResult(ctx context.Context, id string, execution testkube.ExecutionResult) error {
	return nil
}
func (FakeResultRepository) StartExecution(ctx context.Context, id string, startTime time.Time) error {
	return nil
}
func (FakeResultRepository) EndExecution(ctx context.Context, execution testkube.Execution) error {
	return nil
}
