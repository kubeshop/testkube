package triggers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	executorsclientv1 "github.com/kubeshop/testkube-operator/client/executors/v1"
	testsclientv3 "github.com/kubeshop/testkube-operator/client/tests/v3"
	testsuitesv2 "github.com/kubeshop/testkube-operator/client/testsuites/v2"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/result"
	"github.com/kubeshop/testkube/internal/pkg/api/repository/testresult"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/executor/client"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/scheduler"
	"github.com/kubeshop/testkube/pkg/secret"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExecute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockResultRepository := result.NewMockRepository(mockCtrl)
	mockTestResultRepository := testresult.NewMockRepository(mockCtrl)

	mockExecutorsClient := executorsclientv1.NewMockInterface(mockCtrl)
	mockTestsClient := testsclientv3.NewMockInterface(mockCtrl)
	mockTestSuitesClient := testsuitesv2.NewMockInterface(mockCtrl)
	mockSecretClient := secret.NewMockInterface(mockCtrl)

	mockExecutor := client.NewMockExecutor(mockCtrl)

	mockEventEmitter := event.NewEmitter(bus.NewEventBusMock())

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
	mockTestsClient.EXPECT().Get("some-test").Return(&mockTest, nil).Times(2)
	mockNextExecutionNumber := 1
	mockResultRepository.EXPECT().GetNextExecutionNumber(gomock.Any(), "some-test").Return(mockNextExecutionNumber, nil)
	mockExecution := testkube.Execution{Name: "test-execution-1"}
	mockResultRepository.EXPECT().GetByNameAndTest(gomock.Any(), "some-custom-execution-1", "some-test").Return(mockExecution, nil)
	mockSecretUUID := "b524c2f6-6bcf-4178-87c1-1aa2b2abb5dc"
	mockTestsClient.EXPECT().GetCurrentSecretUUID("some-test").Return(mockSecretUUID, nil)
	mockExecutorTypes := "cypress"
	mockExecutorV1 := v1.Executor{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "cypress"},
		Spec: v1.ExecutorSpec{
			Types:            []string{mockExecutorTypes},
			ExecutorType:     "job",
			URI:              "",
			Image:            "cypress",
			Args:             nil,
			Command:          []string{"run"},
			ImagePullSecrets: nil,
			Features:         nil,
			ContentTypes:     nil,
			JobTemplate:      "",
		},
	}
	mockExecutorsClient.EXPECT().GetByType(mockExecutorTypes).Return(&mockExecutorV1, nil)
	mockResultRepository.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
	mockResultRepository.EXPECT().StartExecution(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockSecretClient.EXPECT().Get("some-test-secrets").Return(nil, nil)
	mockExecutionResult := testkube.ExecutionResult{Status: testkube.ExecutionStatusRunning}
	mockExecutor.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(mockExecutionResult, nil)
	mockResultRepository.EXPECT().UpdateResult(gomock.Any(), gomock.Any(), mockExecutionResult).Return(nil)

	rnr := scheduler.NewRunner(
		mockExecutor,
		mockResultRepository,
		mockTestResultRepository,
		mockExecutorsClient,
		mockTestsClient,
		mockTestSuitesClient,
		mockSecretClient,
		mockEventEmitter,
		log.DefaultLogger,
	)
	s := &Service{
		triggerStatus:    make(map[statusKey]*triggerStatus),
		runner:           rnr,
		testsClient:      mockTestsClient,
		testSuitesClient: mockTestSuitesClient,
		logger:           log.DefaultLogger,
	}

	testTrigger := testtriggersv1.TestTrigger{
		ObjectMeta: metav1.ObjectMeta{Namespace: "testkube", Name: "test-trigger-1"},
		Spec: testtriggersv1.TestTriggerSpec{
			Resource:         "pod",
			ResourceSelector: testtriggersv1.TestTriggerSelector{Name: "test-pod"},
			Event:            "created",
			Action:           "run",
			Execution:        "test",
			TestSelector:     testtriggersv1.TestTriggerSelector{Name: "some-test"},
		},
	}

	s.addTrigger(&testTrigger)

	assert.Len(t, s.triggers, 1)
	key := newStatusKey(testTrigger.Namespace, testTrigger.Name)
	_, ok := s.triggerStatus[key]
	assert.True(t, ok)

	err := s.execute(ctx, &testTrigger)
	assert.NoError(t, err)
}
