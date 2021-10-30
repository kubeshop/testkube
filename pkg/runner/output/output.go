package output

import (
	"encoding/json"
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const TypeLogEvent = "event"
const TypeLogLine = "line"
const TypeError = "error"
const TypeResult = "result"

func NewOutputEvent(message string, content interface{}) Output {
	return Output{
		Type:    TypeLogEvent,
		Message: message,
		Content: content,
	}
}

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
	Message string      `json:"message,omitempty"`
	Content interface{} `json:"content,omitempty"`
}

func PrintError(err error) {
	out, _ := json.Marshal(NewOutputError(err))
	fmt.Printf("%s\n", out)
}

func PrintLog(message string) {
	out, _ := json.Marshal(NewOutputLine([]byte(message)))
	fmt.Printf("%s\n", out)
}

func PrintResult(result testkube.ExecutionResult) {
	out, _ := json.Marshal(NewOutputResult(result))
	fmt.Printf("%s\n", out)
}

func PrintEvent(message string, obj ...interface{}) {
	out, _ := json.Marshal(NewOutputEvent(message, obj))
	fmt.Printf("%s\n", out)
}
