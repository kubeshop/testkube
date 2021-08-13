package kubetest

func (r ExecutionResult) Err(err error) ExecutionResult {
	r.ErrorMessage = err.Error()
	return r
}
