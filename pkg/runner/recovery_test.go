package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestHealRecoveredResult(t *testing.T) {
	running := func() *testkube.TestWorkflowResult {
		return &testkube.TestWorkflowResult{
			Status:         common.Ptr(testkube.RUNNING_TestWorkflowStatus),
			Initialization: &testkube.TestWorkflowStepResult{Status: common.Ptr(testkube.RUNNING_TestWorkflowStepStatus)},
			Steps:          map[string]testkube.TestWorkflowStepResult{"a": {Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus)}},
		}
	}
	finished := func() *testkube.TestWorkflowResult {
		return &testkube.TestWorkflowResult{
			Status:         common.Ptr(testkube.ABORTED_TestWorkflowStatus),
			FinishedAt:     time.Now(),
			Initialization: &testkube.TestWorkflowStepResult{Status: common.Ptr(testkube.ABORTED_TestWorkflowStepStatus), ErrorMessage: "The execution has been aborted. (by the user: the pod cannot be scheduled)"},
			Steps:          map[string]testkube.TestWorkflowStepResult{"a": {Status: common.Ptr(testkube.ABORTED_TestWorkflowStepStatus), ErrorMessage: "The execution has been aborted. (by the user)"}},
		}
	}

	tests := []struct {
		name               string
		result             *testkube.TestWorkflowResult
		cause              string
		causeReason        string
		wantStatus         testkube.TestWorkflowStatus
		wantInitialization string
		wantStepStatus     testkube.TestWorkflowStepStatus
		wantStep           string
		wantInitReason     string
	}{
		{
			name:               "writes a cause that differs from the message of the result as the reason",
			result:             running(),
			cause:              "the parent could not pull the image",
			causeReason:        string(testkube.StartReasonImagePullFailed),
			wantStatus:         testkube.ABORTED_TestWorkflowStatus,
			wantInitialization: "The execution has been aborted. (the parent could not pull the image)",
			wantStepStatus:     testkube.SKIPPED_TestWorkflowStepStatus,
			wantStep:           "The execution was aborted before. (the parent could not pull the image)",
			wantInitReason:     string(testkube.StartReasonImagePullFailed),
		},
		{
			name:               "does not change a result that is already finished",
			result:             finished(),
			cause:              "The execution has been aborted. (by the user: the pod cannot be scheduled)",
			wantStatus:         testkube.ABORTED_TestWorkflowStatus,
			wantInitialization: "The execution has been aborted. (by the user: the pod cannot be scheduled)",
			wantStepStatus:     testkube.ABORTED_TestWorkflowStepStatus,
			wantStep:           "The execution has been aborted. (by the user)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healRecoveredResult(tt.result, []testkube.TestWorkflowSignature{{Ref: "a"}}, time.Now(), tt.cause, tt.causeReason)

			assert.Equal(t, tt.wantStatus, *tt.result.Status)
			assert.Equal(t, tt.wantInitialization, tt.result.Initialization.ErrorMessage)
			assert.Equal(t, tt.wantStepStatus, *tt.result.Steps["a"].Status)
			assert.Equal(t, tt.wantStep, tt.result.Steps["a"].ErrorMessage)
			assert.Equal(t, tt.wantInitReason, tt.result.Initialization.ErrorReason)
		})
	}
}

func TestRecordedCause(t *testing.T) {
	sigSequence := []testkube.TestWorkflowSignature{{Ref: "a"}, {Ref: "b"}}
	tests := []struct {
		name       string
		result     *testkube.TestWorkflowResult
		want       string
		wantReason string
	}{
		{
			name: "returns the initialization message first",
			result: &testkube.TestWorkflowResult{
				Initialization: &testkube.TestWorkflowStepResult{ErrorMessage: "the image could not be pulled"},
				Steps:          map[string]testkube.TestWorkflowStepResult{"a": {ErrorMessage: "the step failed"}},
			},
			want: "the image could not be pulled",
		},
		{
			name: "returns the first step message in the order of the signature",
			result: &testkube.TestWorkflowResult{
				Initialization: &testkube.TestWorkflowStepResult{},
				Steps:          map[string]testkube.TestWorkflowStepResult{"b": {ErrorMessage: "second"}, "a": {ErrorMessage: "first"}},
			},
			want: "first",
		},
		{
			name: "skips the message of a step that finished normally",
			result: &testkube.TestWorkflowResult{
				Initialization: &testkube.TestWorkflowStepResult{Status: common.Ptr(testkube.PASSED_TestWorkflowStepStatus)},
				Steps: map[string]testkube.TestWorkflowStepResult{
					"a": {Status: common.Ptr(testkube.FAILED_TestWorkflowStepStatus), ErrorMessage: "1 of 1 executions failed"},
					"b": {Status: common.Ptr(testkube.QUEUED_TestWorkflowStepStatus), ErrorMessage: "the image could not be pulled", ErrorReason: string(testkube.StartReasonImagePullFailed)},
				},
			},
			want:       "the image could not be pulled",
			wantReason: string(testkube.StartReasonImagePullFailed),
		},
		{
			name:   "returns an empty cause without an initialization step or step messages",
			result: &testkube.TestWorkflowResult{Steps: map[string]testkube.TestWorkflowStepResult{"a": {}}},
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, reason := recordedCause(tt.result, sigSequence)
			assert.Equal(t, tt.want, message)
			assert.Equal(t, tt.wantReason, reason)
		})
	}
}
