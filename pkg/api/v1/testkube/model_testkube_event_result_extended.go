package testkube

func NewFailedEventResult(id string, err error) EventResult {
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	return EventResult{
		Error_: errStr,
		Id:     id,
	}
}

func NewSuccessEventResult(id string, result string) EventResult {
	return EventResult{
		Id:     id,
		Result: result,
	}
}

func (l EventResult) Error() string {
	return l.Error_
}

func (l EventResult) WithResult(result string) EventResult {
	l.Result = result
	return l
}
