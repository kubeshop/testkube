package triggers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

func TestService_runExecutionScraper(t *testing.T) {

	t.Run("completed jobs", func(t *testing.T) {

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		ctx, cancel := context.WithTimeout(context.Background(), 310*time.Millisecond)
		defer cancel()

		testWorkflowStatus := testkube.PASSED_TestWorkflowStatus
		mockTestWorkflowResultsRepository := testworkflow.NewMockRepository(mockCtrl)
		mockTestWorkflowExecution := testkube.TestWorkflowExecution{Id: "test-workflow-execution-1", Result: &testkube.TestWorkflowResult{Status: &testWorkflowStatus}}
		mockTestWorkflowResultsRepository.EXPECT().Get(gomock.Any(), "test-workflow-execution-1").Return(mockTestWorkflowExecution, nil)

		statusKey3 := newStatusKey("testkube", "test-trigger-3")
		triggerStatus3 := &triggerStatus{testWorkflowExecutionIDs: []string{"test-workflow-execution-1"}}
		triggerStatusMap := map[statusKey]*triggerStatus{
			statusKey3: triggerStatus3,
		}
		s := &Service{
			triggerStatus:                 triggerStatusMap,
			testWorkflowResultsRepository: mockTestWorkflowResultsRepository,
			scraperInterval:               100 * time.Millisecond,
			logger:                        log.DefaultLogger,
		}

		s.runExecutionScraper(ctx)

		for testTrigger, status := range s.triggerStatus {
			assert.Falsef(t, status.hasActiveTests(), "TestTrigger V1 %s should not have active tests", testTrigger)
		}
	})

	t.Run("active jobs", func(t *testing.T) {

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		ctx, cancel := context.WithTimeout(context.Background(), 310*time.Millisecond)
		defer cancel()

		testWorkflowStatus := testkube.RUNNING_TestWorkflowStatus
		mockTestWorkflowExecution := testkube.TestWorkflowExecution{Id: "test-workflow-execution-1", Result: &testkube.TestWorkflowResult{Status: &testWorkflowStatus}}
		mockTestWorkflowResultsRepository := testworkflow.NewMockRepository(mockCtrl)
		mockTestWorkflowResultsRepository.EXPECT().Get(gomock.Any(), "test-workflow-execution-1").Return(mockTestWorkflowExecution, nil).Times(3)

		statusKey2 := newStatusKey("testkube", "test-trigger-2")
		triggerStatus2 := &triggerStatus{testWorkflowExecutionIDs: []string{"test-workflow-execution-1"}}
		triggerStatusMap := map[statusKey]*triggerStatus{
			statusKey2: triggerStatus2,
		}
		s := &Service{
			triggerStatus:                 triggerStatusMap,
			testWorkflowResultsRepository: mockTestWorkflowResultsRepository,
			scraperInterval:               100 * time.Millisecond,
			logger:                        log.DefaultLogger,
		}

		s.runExecutionScraper(ctx)

		for testTrigger, status := range s.triggerStatus {
			assert.Truef(t, status.hasActiveTests(), "TestTrigger V1 %s should not have finished tests", testTrigger)
		}
	})
}
