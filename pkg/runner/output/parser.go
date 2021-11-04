package output

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func GetLogEntry(b []byte) (out Output, err error) {
	err = json.Unmarshal(b, &out)
	return out, err
}

// GetExecutionResult tries to unmarshal execution result
func GetExecutionResult(b []byte) (is bool, result testkube.ExecutionResult) {
	err := json.Unmarshal(b, &result)
	return err == nil, result
}

// ParseRunnerOutput try to parse possible runner output which is some bunch
// of json stream like
// {"type": "line", "content": "runner execution started  ------------"}
// {"type": "line", "content": "GET /results"}
// {"type": "result", "content": {"id": "2323", "output": "-----"}}
func ParseRunnerOutput(b []byte) (result testkube.ExecutionResult, logs []string, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(b))

	// try to locate execution result should be the last one
	// but there could be some buffers or go routines used so go through whole
	// array too
	for scanner.Scan() {
		b := scanner.Bytes()

		if len(b) < 2 || b[0] != byte('{') {
			// empty or non json line
			continue
		}
		log, err := GetLogEntry(scanner.Bytes())
		if err != nil {
			// try to read in case of some lines which we couldn't parse
			// sometimes we're not able to control all stdout messages from libs
			logs = append(logs, fmt.Sprintf("ERROR can't get log entry: %s, (((%s)))", err, scanner.Text()))
			continue
		}

		switch log.Type {
		case TypeResult:
			result = *log.Result

		case TypeError:
			result = testkube.ExecutionResult{ErrorMessage: log.Message}

		case TypeLogLine:
			if l, ok := log.Content.(string); ok {
				logs = append(logs, l)
			}
		case TypeLogEvent:
			logs = append(logs, log.Message)
		}

	}

	return result, logs, scanner.Err()
}
