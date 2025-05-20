package cronjob

import (
	"context"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	testsuitesv3 "github.com/kubeshop/testkube-operator/api/testsuite/v3"
	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	testsuitesclientv3 "github.com/kubeshop/testkube-operator/pkg/client/testsuites/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

func Test_execuuteTestWorkflow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTestWorkflowsClient := testworkflowclient.NewMockTestWorkflowClient(mockCtrl)
	mockTestWorkflowTemplatesClient := testworkflowtemplateclient.NewMockTestWorkflowTemplateClient(mockCtrl)
	mockTestWorkflowExecutor := testworkflowexecutor.NewMockTestWorkflowExecutor(mockCtrl)

	mockTestWorkflow := testkube.TestWorkflow{Name: "testworkflow"}
	mockTestWorkflowExecutionRequest := &cloud.ScheduleRequest{
		Executions: []*cloud.ScheduleExecution{
			{Selector: &cloud.ScheduleResourceSelector{Name: mockTestWorkflow.Name}},
		},
	}

	executionsCh := make(chan *testkube.TestWorkflowExecution, 1)
	executionsCh <- &testkube.TestWorkflowExecution{}
	close(executionsCh)
	executionsStream := testworkflowexecutor.NewStream(executionsCh)
	mockTestWorkflowExecutor.EXPECT().Execute(ctx, "", mockTestWorkflowExecutionRequest).Return(executionsStream).Times(1)

	scheduler := New(mockTestWorkflowsClient, mockTestWorkflowTemplatesClient, mockTestWorkflowExecutor, log.DefaultLogger)

	scheduler.executeTestWorkflow(ctx, mockTestWorkflow.Name,
		&testkube.TestWorkflowCronJobConfig{Cron: "* * * * * *", Timezone: &testkube.BoxedString{Value: "America/New_York"}})
}

func Test_executeTest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTestWorkflowsClient := testworkflowclient.NewMockTestWorkflowClient(mockCtrl)
	mockTestWorkflowTemplatesClient := testworkflowtemplateclient.NewMockTestWorkflowTemplateClient(mockCtrl)
	mockTestWorkflowExecutor := testworkflowexecutor.NewMockTestWorkflowExecutor(mockCtrl)
	mockTestClient := testsclientv3.NewMockInterface(mockCtrl)

	mockTest := &testsv3.Test{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	executeTestFn := func(ctx context.Context, t testkube.Test, r testkube.ExecutionRequest) (testkube.Execution, error) {
		return testkube.Execution{}, nil
	}

	mockTestClient.EXPECT().Get(mockTest.Name).Return(mockTest, nil).Times(1)

	scheduler := New(mockTestWorkflowsClient, mockTestWorkflowTemplatesClient, mockTestWorkflowExecutor, log.DefaultLogger,
		WithTestClient(mockTestClient), WithExecuteTestFn(executeTestFn))

	err := scheduler.executeTest(ctx, mockTest.Name, "* * * * * *")
	assert.NoError(t, err)
}

func Test_executeTestSuite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTestWorkflowsClient := testworkflowclient.NewMockTestWorkflowClient(mockCtrl)
	mockTestWorkflowTemplatesClient := testworkflowtemplateclient.NewMockTestWorkflowTemplateClient(mockCtrl)
	mockTestWorkflowExecutor := testworkflowexecutor.NewMockTestWorkflowExecutor(mockCtrl)
	mockTestSuiteClient := testsuitesclientv3.NewMockInterface(mockCtrl)

	mockTestSuite := &testsuitesv3.TestSuite{ObjectMeta: metav1.ObjectMeta{Name: "testsuite"}}
	executeTestSuiteFn := func(ctx context.Context, t testkube.TestSuite, r testkube.TestSuiteExecutionRequest) (testkube.TestSuiteExecution, error) {
		return testkube.TestSuiteExecution{}, nil
	}

	mockTestSuiteClient.EXPECT().Get(mockTestSuite.Name).Return(mockTestSuite, nil).Times(1)

	scheduler := New(mockTestWorkflowsClient, mockTestWorkflowTemplatesClient, mockTestWorkflowExecutor, log.DefaultLogger,
		WithTestSuiteClient(mockTestSuiteClient), WithExecuteTestSuiteFn(executeTestSuiteFn))

	err := scheduler.executeTestSuite(ctx, mockTestSuite.Name, "* * * * * *")
	assert.NoError(t, err)
}
