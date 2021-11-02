package testkube

func StatusPtr(status ExecutionStatus) *ExecutionStatus {
	return &status
}

var ExecutionStatusError = StatusPtr(ERROR__ExecutionStatus)
var ExecutionStatusSuccess = StatusPtr(SUCCESS_ExecutionStatus)
var ExecutionStatusQueued = StatusPtr(QUEUED_ExecutionStatus)
var ExecutionStatusPending = StatusPtr(PENDING_ExecutionStatus)
