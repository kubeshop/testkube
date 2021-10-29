package output

import (
	"encoding/json"
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

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

func PrintError(err error) {
	out, _ := json.Marshal(NewOutputError(err))
	fmt.Printf("%s", out)
}

func PrintLog(message string) {
	out, _ := json.Marshal(NewOutputLine([]byte(message)))
	fmt.Printf("%s", out)
}

func PrintResult(result testkube.ExecutionResult) {
	out, _ := json.Marshal(NewOutputResult(result))
	fmt.Printf("%s", out)
}
