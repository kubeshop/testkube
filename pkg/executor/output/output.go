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

func NewOutputEvent(message string) Output {
	return Output{
		Type_:   TypeLogEvent,
		Content: message,
	}
}

func NewOutputLine(content []byte) Output {
	return Output{
		Type_:   TypeLogLine,
		Content: string(content),
	}
}

func NewOutputError(err error) Output {
	return Output{
		Type_:   TypeError,
		Content: string(err.Error()),
	}
}

func NewOutputResult(result testkube.ExecutionResult) Output {
	return Output{
		Type_:  TypeResult,
		Result: &result,
	}
}

type Output testkube.ExecutorOutput

func (out Output) String() string {
	switch out.Type_ {
	case TypeError, TypeLogLine, TypeLogEvent:
		return out.Content
	case TypeResult:
		b, _ := json.Marshal(out.Result)
		return string(b)
	}

	return ""
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
	out, _ := json.Marshal(NewOutputEvent(fmt.Sprintf("%s %v", message, obj)))
	fmt.Printf("%s\n", out)
}
