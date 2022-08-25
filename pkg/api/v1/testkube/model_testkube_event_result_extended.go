package testkube

func NewFailedTestkubeEventResult(id string, err error) TestkubeEventResult {
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	return TestkubeEventResult{
		Error_: errStr,
		Id:     id,
	}
}

func NewSuccessTestkubeEventResult(id string, result string) TestkubeEventResult {
	return TestkubeEventResult{
		Id:     id,
		Result: result,
	}
}

func (l TestkubeEventResult) Error() string {
	return l.Error_
}

func (l TestkubeEventResult) WithResult(result string) TestkubeEventResult {
	l.Result = result
	return l
}
