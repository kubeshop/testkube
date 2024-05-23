package triggers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/executors/v1"
	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	testsourcesv1 "github.com/kubeshop/testkube-operator/pkg/client/testsources/v1"
	testsuiteexecutionsv1 "github.com/kubeshop/testkube-operator/pkg/client/testsuiteexecutions/v1"
	testsuitesv3 "github.com/kubeshop/testkube-operator/pkg/client/testsuites/v3"
	testworkflowsclientv1 "github.com/kubeshop/testkube-operator/pkg/client/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/configmap"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/featureflags"
	"github.com/kubeshop/testkube/pkg/log"
	logsclient "github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/result"
	"github.com/kubeshop/testkube/pkg/repository/testresult"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/kubeshop/testkube/pkg/tcl/testworkflowstcl/testworkflowexecutor"
)

func TestExecute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	metricsHandle := metrics.NewMetrics()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockBus := bus.NewEventBusMock()
	mockResultRepository := result.NewMockRepository(mockCtrl)
	mockTestResultRepository := testresult.NewMockRepository(mockCtrl)

	mockExecutorsClient := executorsclientv1.NewMockInterface(mockCtrl)
	mockTestsClient := testsclientv3.NewMockInterface(mockCtrl)
	mockTestSuitesClient := testsuitesv3.NewMockInterface(mockCtrl)
	mockTestSourcesClient := testsourcesv1.NewMockInterface(mockCtrl)
	mockSecretClient := secret.NewMockInterface(mockCtrl)
	configMapConfig := config.NewMockRepository(mockCtrl)
	mockConfigMapClient := configmap.NewMockInterface(mockCtrl)
	mockTestSuiteExecutionsClient := testsuiteexecutionsv1.NewMockInterface(mockCtrl)

	mockExecutor := client.NewMockExecutor(mockCtrl)

	mockEventEmitter := event.NewEmitter(bus.NewEventBusMock(), "", nil)

	mockTest := testsv3.Test{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "some-test"},
		Spec: testsv3.TestSpec{
			Type_: "cypress",
			ExecutionRequest: &testsv3.ExecutionRequest{
				Name:   "some-custom-execution",
				Number: 1,
				Image:  "test-image",
			},
		},
	}
	mockTestsClient.EXPECT().Get("some-test").Return(&mockTest, nil).AnyTimes()
	var mockNextExecutionNumber int32 = 1
	mockResultRepository.EXPECT().GetNextExecutionNumber(gomock.Any(), "some-test").Return(mockNextExecutionNumber, nil)
	mockExecutionResult := testkube.ExecutionResult{Status: testkube.ExecutionStatusRunning}
	mockExecution := testkube.Execution{Name: "test-execution-1"}
	mockExecution.ExecutionResult = &mockExecutionResult
	mockResultRepository.EXPECT().GetByNameAndTest(gomock.Any(), "some-custom-execution", "some-test").Return(mockExecution, nil)
	mockSecretUUID := "b524c2f6-6bcf-4178-87c1-1aa2b2abb5dc"
	mockTestsClient.EXPECT().GetCurrentSecretUUID("some-test").Return(mockSecretUUID, nil)
	mockExecutorTypes := "cypress"
	mockExecutorV1 := v1.Executor{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "cypress"},
		Spec: v1.ExecutorSpec{
			Types:                  []string{mockExecutorTypes},
			ExecutorType:           "job",
			URI:                    "",
			Image:                  "cypress",
			Args:                   nil,
			Command:                []string{"run"},
			ImagePullSecrets:       nil,
			Features:               nil,
			ContentTypes:           nil,
			JobTemplate:            "",
			JobTemplateReference:   "",
			Meta:                   nil,
			UseDataDirAsWorkingDir: false,
		},
	}
	mockExecutorsClient.EXPECT().GetByType(mockExecutorTypes).Return(&mockExecutorV1, nil).AnyTimes()
	mockResultRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
	mockResultRepository.EXPECT().StartExecution(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockExecutor.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).Return(&mockExecutionResult, nil)
	mockResultRepository.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	mockLogsStream := logsclient.NewMockStream(mockCtrl)

	sched := scheduler.NewScheduler(
		metricsHandle,
		mockExecutor,
		mockExecutor,
		mockResultRepository,
		mockTestResultRepository,
		mockExecutorsClient,
		mockTestsClient,
		mockTestSuitesClient,
		mockTestSourcesClient,
		mockSecretClient,
		mockEventEmitter,
		log.DefaultLogger,
		configMapConfig,
		mockConfigMapClient,
		mockTestSuiteExecutionsClient,
		mockBus,
		"",
		featureflags.FeatureFlags{},
		mockLogsStream,
		"",
		"",
		"",
	)
	s := &Service{
		triggerStatus:    make(map[statusKey]*triggerStatus),
		scheduler:        sched,
		testsClient:      mockTestsClient,
		testSuitesClient: mockTestSuitesClient,
		logger:           log.DefaultLogger,
	}

	status := testtriggersv1.TRUE_TestTriggerConditionStatuses
	testTrigger := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:         "deployment",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-deployment"},
			Event:            "created",
			ConditionSpec: &testtriggersv1.TestTriggerConditionSpec{
				Conditions: []testtriggersv1.TestTriggerCondition{{
					Type_:  "Progressing",
					Status: &status,
					Reason: "NewReplicaSetAvailable",
					Ttl:    60,
				}},
			},
			ProbeSpec: &testtriggersv1.TestTriggerProbeSpec{
				Probes: []testtriggersv1.TestTriggerProbe{{
					Host:    "testkube-api-server",
					Path:    "/health",
					Port:    8088,
					Headers: map[string]string{"X-Token": "12345"},
				}},
			},
			Action:            "run",
			Execution:         "test",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}

	s.addTrigger(&testTrigger)

	key := newStatusKey(testTrigger.Namespace, testTrigger.Name)
	assert.Contains(t, s.triggerStatus, key)

	err := s.execute(ctx, &watcherEvent{}, &testTrigger)
	assert.NoError(t, err)
}

func TestWorkflowExecute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTestWorkflowsClient := testworkflowsclientv1.NewMockInterface(mockCtrl)
	mockTestWorkflowExecutor := testworkflowexecutor.NewMockTestWorkflowExecutor(mockCtrl)

	mockTestWorkflow := testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "some-test"}}
	mockTestWorkflowsClient.EXPECT().Get("some-test").Return(&mockTestWorkflow, nil).AnyTimes()
	mockTestWorkflowExecutionRequest := testkube.TestWorkflowExecutionRequest{
		Config: map[string]string{
			"WATCHER_EVENT_EVENT_TYPE": "",
			"WATCHER_EVENT_NAME":       "",
			"WATCHER_EVENT_NAMESPACE":  "",
			"WATCHER_EVENT_RESOURCE":   "",
		},
	}
	mockTestWorkflowExecution := testkube.TestWorkflowExecution{}
	mockTestWorkflowExecutor.EXPECT().Execute(gomock.Any(), mockTestWorkflow, mockTestWorkflowExecutionRequest).Return(mockTestWorkflowExecution, nil)

	s := &Service{
		triggerStatus:        make(map[statusKey]*triggerStatus),
		testWorkflowsClient:  mockTestWorkflowsClient,
		testWorkflowExecutor: mockTestWorkflowExecutor,
		logger:               log.DefaultLogger,
	}

	status := testtriggersv1.TRUE_TestTriggerConditionStatuses
	testTrigger := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:         "deployment",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-deployment"},
			Event:            "created",
			ConditionSpec: &testtriggersv1.TestTriggerConditionSpec{
				Conditions: []testtriggersv1.TestTriggerCondition{{
					Type_:  "Progressing",
					Status: &status,
					Reason: "NewReplicaSetAvailable",
					Ttl:    60,
				}},
			},
			ProbeSpec: &testtriggersv1.TestTriggerProbeSpec{
				Probes: []testtriggersv1.TestTriggerProbe{{
					Host:    "testkube-api-server",
					Path:    "/health",
					Port:    8088,
					Headers: map[string]string{"X-Token": "12345"},
				}},
			},
			Action:            "run",
			Execution:         "testworkflow",
			ConcurrencyPolicy: "allow",
			TestSelector:      testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}

	s.addTrigger(&testTrigger)

	key := newStatusKey(testTrigger.Namespace, testTrigger.Name)
	assert.Contains(t, s.triggerStatus, key)

	err := s.execute(ctx, &watcherEvent{}, &testTrigger)
	assert.NoError(t, err)
}
