package output

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

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
// {"type": "line", "message": "runner execution started  ------------"}
// {"type": "line", "message": "GET /results"}
// {"type": "result", "result": {"id": "2323", "output": "-----"}}
func ParseRunnerOutput(b []byte) (result testkube.ExecutionResult, logs []string, err error) {
	reader := bufio.NewReader(bytes.NewReader(b))

	// try to locate execution result should be the last one
	// but there could be some buffers or go routines used so go through whole
	// array too
	result.Status = testkube.ExecutionStatusFailed
	for {
		b, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
		if len(b) < 2 || b[0] != byte('{') {
			// empty or non json line
			continue
		}
		log, err := GetLogEntry(b)
		if err != nil {
			// try to read in case of some lines which we couldn't parse
			// sometimes we're not able to control all stdout messages from libs
			logs = append(logs, fmt.Sprintf("ERROR can't get log entry: %s, (((%s)))", err, string(b)))
			continue
		}

		result.Status = testkube.ExecutionStatusPassed
		switch log.Type_ {
		case TypeResult:
			if log.Result != nil {
				result = *log.Result
			}

		case TypeError:
			result = testkube.NewErrorExecutionResult(fmt.Errorf(log.Content))

		case TypeLogEvent, TypeLogLine:
			logs = append(logs, log.Content)
		}

	}

	return result, logs, err
}
