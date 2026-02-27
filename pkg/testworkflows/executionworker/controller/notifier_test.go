package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestShouldPreferCompletionOverAbortedFallback(t *testing.T) {
	t.Parallel()

	assert.True(t, shouldPreferCompletionOverAbortedFallback(
		string(testkube.ABORTED_TestWorkflowStatus),
		DefaultErrorMessage,
		true,
		false,
		true,
		false,
		false,
		false,
	))

	assert.False(t, shouldPreferCompletionOverAbortedFallback(
		string(testkube.CANCELED_TestWorkflowStatus),
		DefaultErrorMessage,
		true,
		false,
		true,
		false,
		false,
		false,
	))

	assert.False(t, shouldPreferCompletionOverAbortedFallback(
		string(testkube.ABORTED_TestWorkflowStatus),
		"explicit runtime error",
		true,
		false,
		true,
		false,
		false,
		false,
	))

	assert.False(t, shouldPreferCompletionOverAbortedFallback(
		string(testkube.ABORTED_TestWorkflowStatus),
		DefaultErrorMessage,
		false,
		false,
		true,
		false,
		false,
		false,
	))

	assert.False(t, shouldPreferCompletionOverAbortedFallback(
		string(testkube.ABORTED_TestWorkflowStatus),
		DefaultErrorMessage,
		true,
		true,
		true,
		false,
		false,
		false,
	))

	assert.False(t, shouldPreferCompletionOverAbortedFallback(
		string(testkube.ABORTED_TestWorkflowStatus),
		DefaultErrorMessage,
		true,
		false,
		false,
		false,
		false,
		false,
	))

	assert.False(t, shouldPreferCompletionOverAbortedFallback(
		string(testkube.ABORTED_TestWorkflowStatus),
		DefaultErrorMessage,
		true,
		false,
		true,
		true,
		false,
		false,
	))

	assert.False(t, shouldPreferCompletionOverAbortedFallback(
		string(testkube.ABORTED_TestWorkflowStatus),
		DefaultErrorMessage,
		true,
		false,
		true,
		false,
		false,
		true,
	))
}

func TestFinalizeUnfinishedStepsForCompletedExecution(t *testing.T) {
	t.Parallel()

	n := &notifier{
		result: testkube.TestWorkflowResult{
			Initialization: &testkube.TestWorkflowStepResult{
				Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
			},
			Steps: map[string]testkube.TestWorkflowStepResult{
				"group": {
					Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus),
				},
				"leaf-passed": {
					Status: common.Ptr(testkube.PASSED_TestWorkflowStepStatus),
				},
				"leaf-running": {
					Status: common.Ptr(testkube.RUNNING_TestWorkflowStepStatus),
				},
			},
		},
		sigSequence: []testkube.TestWorkflowSignature{
			{Ref: constants.InitStepName},
			{
				Ref: "group",
				Children: []testkube.TestWorkflowSignature{
					{Ref: "leaf-passed"},
					{Ref: "leaf-running"},
				},
			},
			{Ref: "leaf-passed"},
			{Ref: "leaf-running"},
		},
	}

	n.finalizeUnfinishedStepsForCompletedExecution()

	assert.Equal(t, testkube.PASSED_TestWorkflowStepStatus, *n.result.Initialization.Status)
	assert.Equal(t, testkube.PASSED_TestWorkflowStepStatus, *n.result.Steps["leaf-passed"].Status)
	assert.Equal(t, testkube.SKIPPED_TestWorkflowStepStatus, *n.result.Steps["leaf-running"].Status)
	assert.Equal(t, testkube.PASSED_TestWorkflowStepStatus, *n.result.Steps["group"].Status)
}
