package cronjob

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsuitesv3 "github.com/kubeshop/testkube/api/testsuite/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/operator/client/common"
	testsuitesclientv3 "github.com/kubeshop/testkube/pkg/operator/client/testsuites/v3"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

func Test_ReconcileTestSuites(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTestWorkflowsClient := testworkflowclient.NewMockTestWorkflowClient(mockCtrl)
	mockTestWorkflowTemplatesClient := testworkflowtemplateclient.NewMockTestWorkflowTemplateClient(mockCtrl)
	mockTestWorkflowExecutor := testworkflowexecutor.NewMockTestWorkflowExecutor(mockCtrl)
	mockTestSuiteClient := testsuitesclientv3.NewMockInterface(mockCtrl)
	mockTestSuiteRESTClient := testsuitesclientv3.NewMockRESTInterface(mockCtrl)

	mockTestSuite := &testsuitesv3.TestSuite{ObjectMeta: metav1.ObjectMeta{Name: "testsuite"}, Spec: testsuitesv3.TestSuiteSpec{Schedule: "* * * * *"}}
	executeTestSuiteFn := func(ctx context.Context, t testkube.TestSuite, r testkube.TestSuiteExecutionRequest) (testkube.TestSuiteExecution, error) {
		return testkube.TestSuiteExecution{}, nil
	}

	mockTestSuiteClient.EXPECT().Get(mockTestSuite.Name).Return(mockTestSuite, nil).AnyTimes()

	result := common.NewWatcher[testsuitesclientv3.Update]()
	mockTestSuiteRESTClient.EXPECT().WatchUpdates(ctx, "", gomock.Any()).Return(result).AnyTimes()
	go func() {
		result.Send(testsuitesclientv3.Update{
			Type:      common.EventTypeCreate,
			Timestamp: time.Now(),
			Resource:  mockTestSuite,
		})

		result.Send(testsuitesclientv3.Update{
			Type:      common.EventTypeUpdate,
			Timestamp: time.Now(),
			Resource:  mockTestSuite,
		})

		result.Send(testsuitesclientv3.Update{
			Type:      common.EventTypeDelete,
			Timestamp: time.Now(),
			Resource:  mockTestSuite,
		})

		result.Close(nil)
		time.Sleep(watcherDelay)
		cancel()
	}()

	scheduler := New(mockTestWorkflowsClient, mockTestWorkflowTemplatesClient, mockTestWorkflowExecutor, log.DefaultLogger,
		WithTestSuiteClient(mockTestSuiteClient), WithExecuteTestSuiteFn(executeTestSuiteFn), WithTestSuiteRESTClient(mockTestSuiteRESTClient))

	err := scheduler.ReconcileTestSuites(ctx)
	assert.EqualError(t, err, context.Canceled.Error())
}
