package output

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

const TypeLogLine = "log"
const TypeError = "error"
const TypeResult = "result"

func NewOutputLine(content []byte) Output {
	return Output{
		Type:    TypeLogLine,
		Content: string(content),
	}
}

func NewOutputError(err error) Output {
	return Output{
		Type:    TypeError,
		Content: string(err.Error()),
		Error:   true,
	}
}

func NewOutputResult(result testkube.ExecutionResult) Output {
	return Output{
		Type:    TypeResult,
		Content: result,
	}
}

type Output struct {
	Type    string      `json:"type,omitempty"`
	Error   bool        `json:"error,omitempty"`
	Content interface{} `json:"content,omitempty"`
}
