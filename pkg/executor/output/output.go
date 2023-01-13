package output

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const TypeLogEvent = "event"
const TypeLogLine = "line"
const TypeError = "error"
const TypeResult = "result"

// NewOutputEvent returns new Output struct of type event
func NewOutputEvent(message string) Output {
	return Output{
		Type_:   TypeLogEvent,
		Content: message,
		Time:    time.Now(),
	}
}

// NewOutputLine returns new Output struct of type line
func NewOutputLine(content []byte) Output {
	return Output{
		Type_:   TypeLogLine,
		Content: string(content),
		Time:    time.Now(),
	}
}

// NewOutputError returns new Output struct of type error
func NewOutputError(err error) Output {
	return Output{
		Type_:   TypeError,
		Content: string(err.Error()),
		Time:    time.Now(),
	}
}

// NewOutputResult returns new Output struct of type result - should be last line in stream as it'll stop listening
func NewOutputResult(result testkube.ExecutionResult) Output {
	return Output{
		Type_:  TypeResult,
		Result: &result,
		Time:   time.Now(),
	}
}

// Output generic json based output data structure
type Output testkube.ExecutorOutput

// String
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

// PrintError - prints error as output json
func PrintError(w io.Writer, err error) {
	out, _ := json.Marshal(NewOutputError(err))
	fmt.Fprintf(w, "%s\n", out)
}

// PrintLog - prints log line as output json
func PrintLog(message string) {
	out, _ := json.Marshal(NewOutputLine([]byte(message)))
	fmt.Printf("%s\n", out)
}

// PrintResult - prints result as output json
func PrintResult(result testkube.ExecutionResult) {
	out, _ := json.Marshal(NewOutputResult(result))
	fmt.Printf("%s\n", out)
}

// PrintEvent - prints event as output json
func PrintEvent(message string, obj ...interface{}) {
	out, _ := json.Marshal(NewOutputEvent(fmt.Sprintf("%s %v", message, obj)))
	fmt.Printf("%s\n", out)
}
