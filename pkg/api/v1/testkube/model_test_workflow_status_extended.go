package testkube

import (
	"fmt"
	"slices"
	"strings"
)

// TestWorkflowStatuses is an array of TestWorkflowStatus
type TestWorkflowStatuses []TestWorkflowStatus

// ToMap generates map from TestWorkflowStatuses
func (statuses TestWorkflowStatuses) ToMap() map[TestWorkflowStatus]struct{} {
	statusMap := map[TestWorkflowStatus]struct{}{}
	for _, status := range statuses {
		statusMap[status] = struct{}{}
	}

	return statusMap
}

// ParseTestWorkflowStatusList parse a list of workflow execution statuses from string
func ParseTestWorkflowStatusList(source, separator string) (statusList TestWorkflowStatuses, err error) {
	statusMap := map[TestWorkflowStatus]struct{}{
		QUEUED_TestWorkflowStatus:     {},
		ASSIGNED_TestWorkflowStatus:   {},
		STARTING_TestWorkflowStatus:   {},
		SCHEDULING_TestWorkflowStatus: {},
		RUNNING_TestWorkflowStatus:    {},
		PAUSING_TestWorkflowStatus:    {},
		PAUSED_TestWorkflowStatus:     {},
		RESUMING_TestWorkflowStatus:   {},
		PASSED_TestWorkflowStatus:     {},
		FAILED_TestWorkflowStatus:     {},
		STOPPING_TestWorkflowStatus:   {},
		ABORTED_TestWorkflowStatus:    {},
		CANCELED_TestWorkflowStatus:   {},
	}

	if source == "" {
		return nil, nil
	}

	values := strings.Split(source, separator)
	for _, value := range values {
		status := TestWorkflowStatus(value)
		if _, ok := statusMap[status]; ok {
			statusList = append(statusList, status)
		} else {
			return nil, fmt.Errorf("unknown test workflow execution status %v", status)
		}
	}

	return statusList, nil
}

func TestWorkflowStatusString(ptr *TestWorkflowStatus) string {
	return string(*ptr)
}

// TestWorkflowExecutingStatus defines all statuses that may be applied to an in-flight execution.
// This is logically the inverse of TestWorkflowTerminalStatus
var TestWorkflowExecutingStatus = []TestWorkflowStatus{
	QUEUED_TestWorkflowStatus,
	ASSIGNED_TestWorkflowStatus,
	STARTING_TestWorkflowStatus,
	SCHEDULING_TestWorkflowStatus,
	RUNNING_TestWorkflowStatus,
	PAUSING_TestWorkflowStatus,
	PAUSED_TestWorkflowStatus,
	RESUMING_TestWorkflowStatus,
}

// TestWorkflowStoppableStatus defines statuses from which it is permitted to
// transition an Execution to a STOPPING, ABORTED, or CANCELLED state.
var TestWorkflowStoppableStatus = []TestWorkflowStatus{
	QUEUED_TestWorkflowStatus,
	ASSIGNED_TestWorkflowStatus,
	STARTING_TestWorkflowStatus,
	SCHEDULING_TestWorkflowStatus,
	RUNNING_TestWorkflowStatus,
	PAUSED_TestWorkflowStatus,
	RESUMING_TestWorkflowStatus,
}

// TestWorkflowTerminalStatus defines all terminal (final state) statuses for a Test Workflow.
var TestWorkflowTerminalStatus = []TestWorkflowStatus{
	PASSED_TestWorkflowStatus,
	FAILED_TestWorkflowStatus,
	ABORTED_TestWorkflowStatus,
	CANCELED_TestWorkflowStatus,
}

// TestWorkflowOngoingStatus are all statuses that are neither queued nor terminated.
var TestWorkflowOngoingStatus = []TestWorkflowStatus{
	ASSIGNED_TestWorkflowStatus,
	STARTING_TestWorkflowStatus,
	SCHEDULING_TestWorkflowStatus,
	RUNNING_TestWorkflowStatus,
	PAUSING_TestWorkflowStatus,
	PAUSED_TestWorkflowStatus,
	RESUMING_TestWorkflowStatus,
	STOPPING_TestWorkflowStatus,
}

func (s TestWorkflowStatus) Finished() bool {
	return s != "" && slices.Contains(TestWorkflowTerminalStatus, s)
}
