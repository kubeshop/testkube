package output

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	TypeLogEvent     = "event"
	TypeLogLine      = "line"
	TypeError        = "error"
	TypeParsingError = "parsing-error"
	TypeResult       = "result"
	TypeUnknown      = "unknown"
)

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

// Output generic json based output data structure
type Output testkube.ExecutorOutput

// String
func (out Output) String() string {
	switch out.Type_ {
	case TypeError, TypeParsingError, TypeLogLine, TypeLogEvent:
		return out.Content
	case TypeResult:
		b, _ := json.Marshal(out.Result)
		return string(b)
	}

	return ""
}

// PrintLog - prints log line as output json
func PrintLog(message string) {
	out, _ := json.Marshal(NewOutputLine([]byte(message)))
	fmt.Printf("%s\n", out)
}

// PrintLogf - prints log line as output json and supports sprintf formatting
func PrintLogf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	out, _ := json.Marshal(NewOutputLine([]byte(message)))
	fmt.Printf("%s\n", out)
}

// PrintEvent - prints event as output json
func PrintEvent(message string, obj ...interface{}) {
	out, _ := json.Marshal(NewOutputEvent(fmt.Sprintf("%s %v", message, obj)))
	fmt.Printf("%s\n", out)
}
