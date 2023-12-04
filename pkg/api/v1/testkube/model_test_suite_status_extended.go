package testkube

import (
	"fmt"
	"strings"
)

func TestSuiteExecutionStatusPtr(status TestSuiteExecutionStatus) *TestSuiteExecutionStatus {
	return &status
}

var TestSuiteExecutionStatusFailed = TestSuiteExecutionStatusPtr(FAILED_TestSuiteExecutionStatus)
var TestSuiteExecutionStatusPassed = TestSuiteExecutionStatusPtr(PASSED_TestSuiteExecutionStatus)
var TestSuiteExecutionStatusQueued = TestSuiteExecutionStatusPtr(QUEUED_TestSuiteExecutionStatus)
var TestSuiteExecutionStatusRunning = TestSuiteExecutionStatusPtr(RUNNING_TestSuiteExecutionStatus)
var TestSuiteExecutionStatusAborting = TestSuiteExecutionStatusPtr(ABORTING_TestSuiteExecutionStatus)
var TestSuiteExecutionStatusAborted = TestSuiteExecutionStatusPtr(ABORTED_TestSuiteExecutionStatus)
var TestSuiteExecutionStatusTimeout = TestSuiteExecutionStatusPtr(TIMEOUT_TestSuiteExecutionStatus)

// TestSuiteExecutionStatuses is an array of TestSuiteExecutionStatus
type TestSuiteExecutionStatuses []TestSuiteExecutionStatus

// ToMap generates map from TestSuiteExecutionStatuses
func (statuses TestSuiteExecutionStatuses) ToMap() map[TestSuiteExecutionStatus]struct{} {
	statusMap := map[TestSuiteExecutionStatus]struct{}{}
	for _, status := range statuses {
		statusMap[status] = struct{}{}
	}

	return statusMap
}

// ParseTestSuiteExecutionStatusList parse a list of test suite execution statuses from string
func ParseTestSuiteExecutionStatusList(source, separator string) (statusList TestSuiteExecutionStatuses, err error) {
	statusMap := map[TestSuiteExecutionStatus]struct{}{
		FAILED_TestSuiteExecutionStatus:  {},
		PASSED_TestSuiteExecutionStatus:  {},
		QUEUED_TestSuiteExecutionStatus:  {},
		RUNNING_TestSuiteExecutionStatus: {},
	}

	if source == "" {
		return nil, nil
	}

	values := strings.Split(source, separator)
	for _, value := range values {
		status := TestSuiteExecutionStatus(value)
		if _, ok := statusMap[status]; ok {
			statusList = append(statusList, status)
		} else {
			return nil, fmt.Errorf("unknown test suite execution status %v", status)
		}
	}

	return statusList, nil
}

func TestSuiteExecutionStatusString(ptr *TestSuiteExecutionStatus) string {
	return string(*ptr)
}
