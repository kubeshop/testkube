package cronjob

import (
	"context"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowclient"
	"github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient"
	"github.com/kubeshop/testkube/pkg/repository/channels"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowexecutor"
)

func Test_ReconcileTestWorkflow(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTestWorkflowsClient := testworkflowclient.NewMockTestWorkflowClient(mockCtrl)
	mockTestWorkflowTemplatesClient := testworkflowtemplateclient.NewMockTestWorkflowTemplateClient(mockCtrl)
	mockTestWorkflowExecutor := testworkflowexecutor.NewMockTestWorkflowExecutor(mockCtrl)

	mockTestWorkflow := &testkube.TestWorkflow{Name: "testworkflow", Spec: &testkube.TestWorkflowSpec{
		Events: []testkube.TestWorkflowEvent{
			{
				Cronjob: &testkube.TestWorkflowCronJobConfig{
					Cron: "* * * * *",
				},
			},
		}}}
	mockTestWorkflowExecutionRequest := &cloud.ScheduleRequest{
		Executions: []*cloud.ScheduleExecution{
			{Selector: &cloud.ScheduleResourceSelector{Name: mockTestWorkflow.Name}},
		},
	}

	executionsCh := make(chan *testkube.TestWorkflowExecution, 1)
	executionsCh <- &testkube.TestWorkflowExecution{}
	close(executionsCh)
	executionsStream := testworkflowexecutor.NewStream(executionsCh)
	mockTestWorkflowExecutor.EXPECT().Execute(ctx, "", mockTestWorkflowExecutionRequest).Return(executionsStream).AnyTimes()

	result := channels.NewWatcher[testworkflowclient.Update]()
	mockTestWorkflowsClient.EXPECT().WatchUpdates(ctx, "", gomock.Any()).Return(result).AnyTimes()
	go func() {
		result.Send(testworkflowclient.Update{
			Type:      testworkflowclient.EventTypeCreate,
			Timestamp: time.Now(),
			Resource:  mockTestWorkflow,
		})

		result.Send(testworkflowclient.Update{
			Type:      testworkflowclient.EventTypeUpdate,
			Timestamp: time.Now(),
			Resource:  mockTestWorkflow,
		})

		result.Send(testworkflowclient.Update{
			Type:      testworkflowclient.EventTypeDelete,
			Timestamp: time.Now(),
			Resource:  mockTestWorkflow,
		})

		result.Close(nil)
		time.Sleep(watcherDelay)
		cancel()
	}()

	scheduler := New(mockTestWorkflowsClient, mockTestWorkflowTemplatesClient, mockTestWorkflowExecutor, log.DefaultLogger)

	err := scheduler.ReconcileTestWorkflows(ctx)
	assert.EqualError(t, err, context.Canceled.Error())
}
