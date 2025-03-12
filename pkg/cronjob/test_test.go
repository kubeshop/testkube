package cronjob

import (
	"context"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	"github.com/kubeshop/testkube-operator/pkg/client/common"
	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

func Test_ReconcileTests(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTestWorkflowsClient := testworkflowclient.NewMockTestWorkflowClient(mockCtrl)
	mockTestWorkflowTemplatesClient := testworkflowtemplateclient.NewMockTestWorkflowTemplateClient(mockCtrl)
	mockTestWorkflowExecutor := testworkflowexecutor.NewMockTestWorkflowExecutor(mockCtrl)
	mockTestClient := testsclientv3.NewMockInterface(mockCtrl)
	mockTestRESTClient := testsclientv3.NewMockRESTInterface(mockCtrl)

	mockTest := &testsv3.Test{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Spec: testsv3.TestSpec{Schedule: "* * * * *"}}
	executeTestFn := func(ctx context.Context, t testkube.Test, r testkube.ExecutionRequest) (testkube.Execution, error) {
		return testkube.Execution{}, nil
	}

	mockTestClient.EXPECT().Get(mockTest.Name).Return(mockTest, nil).AnyTimes()

	result := common.NewWatcher[testsclientv3.Update]()
	mockTestRESTClient.EXPECT().WatchUpdates(ctx, "", gomock.Any()).Return(result).AnyTimes()
	go func() {
		result.Send(testsclientv3.Update{
			Type:      common.EventTypeCreate,
			Timestamp: time.Now(),
			Resource:  mockTest,
		})

		result.Send(testsclientv3.Update{
			Type:      common.EventTypeUpdate,
			Timestamp: time.Now(),
			Resource:  mockTest,
		})

		result.Send(testsclientv3.Update{
			Type:      common.EventTypeDelete,
			Timestamp: time.Now(),
			Resource:  mockTest,
		})

		result.Close(nil)
		time.Sleep(watcherDelay)
		cancel()
	}()

	scheduler := New(mockTestWorkflowsClient, mockTestWorkflowTemplatesClient, mockTestWorkflowExecutor, log.DefaultLogger,
		WithTestClient(mockTestClient), WithExecuteTestFn(executeTestFn), WithTestRESTClient(mockTestRESTClient))

	err := scheduler.ReconcileTests(ctx)
	assert.EqualError(t, err, context.Canceled.Error())
}
