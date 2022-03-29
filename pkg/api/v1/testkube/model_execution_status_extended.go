package testkube

func StatusPtr(status ExecutionStatus) *ExecutionStatus {
	return &status
}

var ExecutionStatusFailed = StatusPtr(FAILED_ExecutionStatus)
var ExecutionStatusPassed = StatusPtr(PASSED_ExecutionStatus)
var ExecutionStatusQueued = StatusPtr(QUEUED_ExecutionStatus)
var ExecutionStatusRunning = StatusPtr(RUNNING_ExecutionStatus)
