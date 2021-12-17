package testkube

func TestStatusPtr(status TestStatus) *TestStatus {
	return &status
}

var TestStatusError = TestStatusPtr(ERROR__TestStatus)
var TestStatusSuccess = TestStatusPtr(SUCCESS_TestStatus)
var TestStatusQueued = TestStatusPtr(QUEUED_TestStatus)
var TestStatusPending = TestStatusPtr(PENDING_TestStatus)
