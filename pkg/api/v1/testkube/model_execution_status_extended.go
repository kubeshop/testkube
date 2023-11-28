package testkube

import (
	"fmt"
	"strings"
)

func StatusPtr(status ExecutionStatus) *ExecutionStatus {
	return &status
}

var (
	ExecutionStatusFailed  = StatusPtr(FAILED_ExecutionStatus)
	ExecutionStatusPassed  = StatusPtr(PASSED_ExecutionStatus)
	ExecutionStatusQueued  = StatusPtr(QUEUED_ExecutionStatus)
	ExecutionStatusRunning = StatusPtr(RUNNING_ExecutionStatus)
	ExecutionStatusAborted = StatusPtr(ABORTED_ExecutionStatus)
	ExecutionStatusTimeout = StatusPtr(TIMEOUT_ExecutionStatus)
)

// ExecutionStatuses is an array of ExecutionStatus
type ExecutionStatuses []ExecutionStatus

// ToMap generates map from ExecutionStatuses
func (statuses ExecutionStatuses) ToMap() map[ExecutionStatus]struct{} {
	statusMap := map[ExecutionStatus]struct{}{}
	for _, status := range statuses {
		statusMap[status] = struct{}{}
	}

	return statusMap
}

// ParseExecutionStatusList parse a list of execution statuses from string
func ParseExecutionStatusList(source, separator string) (statusList ExecutionStatuses, err error) {
	statusMap := map[ExecutionStatus]struct{}{
		FAILED_ExecutionStatus:  {},
		PASSED_ExecutionStatus:  {},
		QUEUED_ExecutionStatus:  {},
		RUNNING_ExecutionStatus: {},
		ABORTED_ExecutionStatus: {},
		TIMEOUT_ExecutionStatus: {},
	}

	if source == "" {
		return nil, nil
	}

	values := strings.Split(source, separator)
	for _, value := range values {
		status := ExecutionStatus(value)
		if _, ok := statusMap[status]; ok {
			statusList = append(statusList, status)
		} else {
			return nil, fmt.Errorf("unknown execution status %v", status)
		}
	}

	return statusList, nil
}

func ExecutionStatusString(ptr *ExecutionStatus) string {
	return string(*ptr)
}
