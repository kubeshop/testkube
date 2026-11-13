package testkube

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldRetryExecution(t *testing.T) {
	details := func(typ StatusDetailsType, reason StopReason) *TestWorkflowStatusDetails {
		return &TestWorkflowStatusDetails{Type_: string(typ), Reason: string(reason)}
	}
	nodeShutdown := details(StatusDetailsTypeExecutionFailure, StopReasonNodeShutdown)

	tests := []struct {
		name string
		in   ExecutionRetryInput
		want bool
	}{
		{"node-shutdown retries by default", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Attempt: 1}, true},
		{"evicted retries", ExecutionRetryInput{Count: 1, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonEvicted), Attempt: 1}, true},
		{"preempted retries", ExecutionRetryInput{Count: 1, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonPreempted), Attempt: 1}, true},
		{"unschedulable retries although init-failure", ExecutionRetryInput{Count: 1, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeInitFailure, StopReasonUnschedulable), Attempt: 1}, true},
		{"last allowed attempt still retries", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Attempt: 3}, true},
		{"stops after count", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Attempt: 4}, false},
		{"no policy", ExecutionRetryInput{Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Attempt: 1}, false},
		{"step failure never retries", ExecutionRetryInput{Count: 3, On: []string{"infrastructure", "timeout", "oom"}, Status: FAILED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeStepFailure, StopReasonExitCode), Attempt: 1}, false},
		{"init failure never retries", ExecutionRetryInput{Count: 3, On: []string{"infrastructure", "timeout", "oom"}, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeInitFailure, StopReasonConfigMissing), Attempt: 1}, false},
		{"user cancel never retries", ExecutionRetryInput{Count: 3, On: []string{"infrastructure"}, Status: CANCELED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeUserCancel, StopReasonUserCancel), Attempt: 1}, false},
		{"a person's stop with an infra reason never retries", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeUserCancel, StopReasonNodeShutdown), Attempt: 1}, false},
		{"passed never retries", ExecutionRetryInput{Count: 3, Status: PASSED_TestWorkflowStatus, StatusDetails: nodeShutdown, Attempt: 1}, false},
		{"running never retries", ExecutionRetryInput{Count: 3, Status: RUNNING_TestWorkflowStatus, StatusDetails: nodeShutdown, Attempt: 1}, false},
		{"no details never retries", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, Attempt: 1}, false},
		{"oom is opt-in", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonOOMKilled), Attempt: 1}, false},
		{"oom retries when listed", ExecutionRetryInput{Count: 3, On: []string{"oom"}, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonOOMKilled), Attempt: 1}, true},
		{"timeout is opt-in", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonExecutionTimeout), Attempt: 1}, false},
		{"timeout retries when listed", ExecutionRetryInput{Count: 3, On: []string{"timeout"}, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonExecutionTimeout), Attempt: 1}, true},
		{"listing only oom drops the infrastructure default", ExecutionRetryInput{Count: 3, On: []string{"oom"}, Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Attempt: 1}, false},
		{"unknown cause retries nothing", ExecutionRetryInput{Count: 3, On: []string{"bogus"}, Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Attempt: 1}, false},
		{"unknown reason (system abort without reason) is skipped", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeUnknown, StopReasonUnknown), Attempt: 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShouldRetryExecution(tt.in))
		})
	}
}
