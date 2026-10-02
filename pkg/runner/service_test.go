package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
)

// TestService_Reattach checks that the reattach path saves the healed result.
func TestService_Reattach(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := controlplaneclient.NewMockClient(ctrl)
	runnerMock := NewMockRunner(ctrl)

	client.EXPECT().
		GetRunnerOngoingExecutions(gomock.Any()).
		Return([]*cloud.UnfinishedExecution{{EnvironmentId: "env-1", Id: "exec-1"}}, nil)
	runnerMock.EXPECT().
		Monitor(gomock.Any(), "org-1", "env-1", "exec-1").
		Return(registry.ErrResourceNotFound)
	client.EXPECT().
		GetExecution(gomock.Any(), "env-1", "exec-1").
		Return(&testkube.TestWorkflowExecution{
			Id:          "exec-1",
			ScheduledAt: time.Now(),
			Signature:   []testkube.TestWorkflowSignature{{Ref: "a"}},
			Result: &testkube.TestWorkflowResult{
				Status:         common.Ptr(testkube.RUNNING_TestWorkflowStatus),
				Initialization: &testkube.TestWorkflowStepResult{Status: common.Ptr(testkube.RUNNING_TestWorkflowStepStatus), ErrorMessage: "the pod cannot be scheduled"},
				Steps:          map[string]testkube.TestWorkflowStepResult{"a": {Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus)}},
			},
		}, nil)
	saved := make(chan *testkube.TestWorkflowResult, 1)
	client.EXPECT().
		FinishExecutionResult(gomock.Any(), "env-1", "exec-1", gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, result *testkube.TestWorkflowResult) error {
			saved <- result
			return nil
		})

	s := &service{
		logger:     zap.NewNop().Sugar(),
		client:     client,
		runner:     runnerMock,
		proContext: config.ProContext{OrgID: "org-1"},
	}
	require.NoError(t, s.reattach(context.Background()))

	select {
	case result := <-saved:
		assert.Equal(t, testkube.ABORTED_TestWorkflowStatus, *result.Status)
		assert.Equal(t, "The execution has been aborted. (the pod cannot be scheduled)", result.Initialization.ErrorMessage)
	case <-time.After(5 * time.Second):
		t.Fatal("the reattach path did not save the result")
	}
}
