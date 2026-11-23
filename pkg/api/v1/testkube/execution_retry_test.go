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
		{"node-shutdown retries by default", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Retries: 0}, true},
		{"evicted retries", ExecutionRetryInput{Count: 1, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonEvicted), Retries: 0}, true},
		{"preempted retries", ExecutionRetryInput{Count: 1, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonPreempted), Retries: 0}, true},
		{"unschedulable retries although init-failure", ExecutionRetryInput{Count: 1, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeInitFailure, StopReasonUnschedulable), Retries: 0}, true},
		{"last allowed retry is still made", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Retries: 2}, true},
		{"stops once count retries are made", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Retries: 3}, false},
		{"a manual rerun starts a fresh budget", ExecutionRetryInput{Count: 1, Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Retries: 0}, true},
		{"no policy", ExecutionRetryInput{Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Retries: 0}, false},
		{"step failure never retries", ExecutionRetryInput{Count: 3, On: []string{"infrastructure", "timeout", "oom"}, Status: FAILED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeStepFailure, StopReasonExitCode), Retries: 0}, false},
		{"init failure never retries", ExecutionRetryInput{Count: 3, On: []string{"infrastructure", "timeout", "oom"}, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeInitFailure, StopReasonConfigMissing), Retries: 0}, false},
		{"user cancel never retries", ExecutionRetryInput{Count: 3, On: []string{"infrastructure"}, Status: CANCELED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeUserCancel, StopReasonUserCancel), Retries: 0}, false},
		{"a person's stop with an infra reason never retries", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeUserCancel, StopReasonNodeShutdown), Retries: 0}, false},
		{"passed never retries", ExecutionRetryInput{Count: 3, Status: PASSED_TestWorkflowStatus, StatusDetails: nodeShutdown, Retries: 0}, false},
		{"running never retries", ExecutionRetryInput{Count: 3, Status: RUNNING_TestWorkflowStatus, StatusDetails: nodeShutdown, Retries: 0}, false},
		{"no details never retries", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, Retries: 0}, false},
		{"oom is opt-in", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonOOMKilled), Retries: 0}, false},
		{"oom retries when listed", ExecutionRetryInput{Count: 3, On: []string{"oom"}, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonOOMKilled), Retries: 0}, true},
		{"timeout is opt-in", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonExecutionTimeout), Retries: 0}, false},
		{"timeout retries when listed", ExecutionRetryInput{Count: 3, On: []string{"timeout"}, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeExecutionFailure, StopReasonExecutionTimeout), Retries: 0}, true},
		{"listing only oom drops the infrastructure default", ExecutionRetryInput{Count: 3, On: []string{"oom"}, Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Retries: 0}, false},
		{"unknown cause retries nothing", ExecutionRetryInput{Count: 3, On: []string{"bogus"}, Status: ABORTED_TestWorkflowStatus, StatusDetails: nodeShutdown, Retries: 0}, false},
		{"unknown reason (system abort without reason) is skipped", ExecutionRetryInput{Count: 3, Status: ABORTED_TestWorkflowStatus, StatusDetails: details(StatusDetailsTypeUnknown, StopReasonUnknown), Retries: 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShouldRetryExecution(tt.in))
		})
	}
}
