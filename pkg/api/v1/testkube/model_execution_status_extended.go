package testkube

func StatusPtr(status ExecutionStatus) *ExecutionStatus {
	return &status
}

var ExecutionStatusError = StatusPtr(FAILED_ExecutionStatus)
var ExecutionStatusSuccess = StatusPtr(PASSED_ExecutionStatus)
var ExecutionStatusQueued = StatusPtr(QUEUED_ExecutionStatus)
var ExecutionStatusPending = StatusPtr(RUNNING_ExecutionStatus)
