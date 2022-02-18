package testkube

// test execution request body
func ExecutorStartRequestToExecution(request ExecutorStartRequest) Execution {
	return Execution{
		Id:       request.Id,
		Name:     request.Name,
		TestType: request.Type_,
		Params:   request.Params,
		Content:  request.Content,
	}
}
