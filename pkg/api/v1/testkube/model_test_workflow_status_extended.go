package testkube

import (
	"fmt"
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
		ABORTED_TestWorkflowStatus: {},
		FAILED_TestWorkflowStatus:  {},
		PASSED_TestWorkflowStatus:  {},
		QUEUED_TestWorkflowStatus:  {},
		RUNNING_TestWorkflowStatus: {},
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
