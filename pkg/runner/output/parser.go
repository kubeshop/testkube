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
// {"type": "line", "message": "runner execution started  ------------"}
// {"type": "line", "message": "GET /results"}
// {"type": "result", "result": {"id": "2323", "output": "-----"}}
func ParseRunnerOutput(b []byte) (result testkube.ExecutionResult, logs []string, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(b))

	// set default bufio scanner buffer (to limit bufio.Scanner: token too long errors on very long lines)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

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

		switch log.Type_ {
		case TypeResult:
			if log.Result != nil {
				result = *log.Result
			}

		case TypeError:
			result = testkube.ExecutionResult{ErrorMessage: log.Content}

		case TypeLogEvent, TypeLogLine:
			logs = append(logs, log.Content)
		}

	}

	return result, logs, scanner.Err()
}
