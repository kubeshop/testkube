package output

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/utils"
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
	// The latest logline will contain either the Result or the last error
	// See: pkg/executor/agent/agent.go: func Run(r runner.Runner, args []string)
	lastLog := Output{
		Time: time.Time{},
	}

	result.Status = testkube.ExecutionStatusFailed
	for {
		b, err := utils.ReadLongLine(reader)
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
		if log.Type_ == TypeLogEvent || log.Type_ == TypeLogLine || log.Type_ == TypeError {
			logs = append(logs, log.Content)
		}
		if log.Time.After(lastLog.Time) {
			lastLog = log
		}
	}

	if lastLog.Time.IsZero() {
		result.Err(fmt.Errorf("no usable logs were found, faulty logs: %v", logs))
		return result, logs, nil
	}

	switch lastLog.Type_ {
	case TypeResult:
		if lastLog.Result != nil {
			result = *lastLog.Result
		}
	case TypeError:
		result = testkube.NewErrorExecutionResult(fmt.Errorf(lastLog.Content))
	}

	return result, logs, err
}
