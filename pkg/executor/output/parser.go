package output

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/utils"
)

func GetLogEntry(b []byte) (out Output, err error) {
	r := bytes.NewReader(b)
	dec := json.NewDecoder(r)
	err = dec.Decode(&out)
	return out, err
}

// ParseRunnerOutput goes over the raw logs in b and parses possible runner output
// The input is a json stream of the form
// {"type": "line", "message": "runner execution started  ------------", "time": "..."}
// {"type": "line", "message": "GET /results", "time": "..."}
// {"type": "result", "result": {"id": "2323", "output": "-----"}, "time": "..."}
func ParseRunnerOutput(b []byte) (*testkube.ExecutionResult, error) {
	result := &testkube.ExecutionResult{}
	if len(b) == 0 {
		errMessage := "no logs found"
		result.Output = errMessage
		return result.Err(errors.New(errMessage)), nil
	}
	logs, err := parseLogs(b)
	if err != nil {
		err := fmt.Errorf("could not parse logs \"%s\": %v", b, err.Error())
		result.Output = err.Error()
		result.Err(err)
		return result.Err(err), err
	}

	log := getDecidingLogLine(logs)
	if log == nil {
		result.Err(errors.New("no logs found"))
		return result, nil
	}

	switch log.Type_ {
	case TypeResult:
		if log.Result != nil {
			result = log.Result
			break
		}
		result.Err(errors.New("found result log with no content"))
	case TypeError:
		result.Err(fmt.Errorf(log.Content))
	default:
		result.Err(fmt.Errorf("wrong log type was found as last log: %v", log))
	}
	result.Output = sanitizeLogs(logs)

	return result, nil
}

// sanitizeLogs creates a human-readable string from a list of Outputs
func sanitizeLogs(logs []Output) string {
	var sb strings.Builder
	for _, l := range logs {
		sb.WriteString(fmt.Sprintf("%s\n", l.Content))
	}
	return sb.String()
}

// parseLogs gets a list of Outputs from raw logs
func parseLogs(b []byte) ([]Output, error) {
	logs := []Output{}
	reader := bufio.NewReader(bytes.NewReader(b))

	for {
		b, err := utils.ReadLongLine(reader)
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return logs, fmt.Errorf("could not read line: %w", err)
		}
		if len(b) < 2 || b[0] != byte('{') {
			// empty or non json line
			continue
		}
		log, err := GetLogEntry(b)
		if err != nil {
			// try to read in case of some lines which we couldn't parse
			// sometimes we're not able to control all stdout messages from libs
			logs = append(logs, Output{
				Type_:   TypeError,
				Content: fmt.Sprintf("ERROR can't get log entry: %s, (((%s)))", err, string(b)),
			})
			continue
		}
		if log.Type_ == TypeResult {
			if log.Result == nil {
				logs = append(logs, Output{
					Type_:   TypeError,
					Content: fmt.Sprintf("ERROR empty result entry: %s, (((%s)))", err, string(b)),
				})
				continue
			}
			message := getResultMessage(*log.Result)
			logs = append(logs, Output{
				Type_:   TypeResult,
				Content: message,
				Result:  log.Result,
			})
			continue
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// getDecidingLogLine returns the log line of type result
// if there is no log line of type result it will return the last log based on time
// if there are no timestamps, it will return the last error log from the list,
// if there are no errors, the last log line
// The latest logline will contain either the Result, the last error or the last log
// See: pkg/executor/agent/agent.go: func Run(r runner.Runner, args []string)
func getDecidingLogLine(logs []Output) *Output {
	if len(logs) == 0 {
		return nil
	}
	resultLog := Output{
		Type_: TypeLogLine,
		Time:  time.Time{},
	}

	for _, log := range logs {
		if log.Type_ == TypeResult && log.Result.IsRunning() {
			// this is the result of the init-container on success, let's ignore it
			continue
		}

		if moreSevere(log.Type_, resultLog.Type_) {
			resultLog = log
			continue
		}

		if sameSeverity(log.Type_, resultLog.Type_) {
			if log.Time.Before(resultLog.Time) {
				continue
			}
			resultLog = log
		}
	}
	if resultLog.Content == "" {
		resultLog = logs[len(logs)-1]
	}

	return &resultLog
}

// getResultMessage returns a message from the result regardless of its type
func getResultMessage(result testkube.ExecutionResult) string {
	if result.IsFailed() {
		return result.ErrorMessage
	}
	if result.IsPassed() {
		return result.Output
	}

	return fmt.Sprintf("%v", result.Status)
}

// sameSeverity decides if a and b are of the same severity type
func sameSeverity(a string, b string) bool {
	return a == b
}

// moreSevere decides if a is more severe than b
func moreSevere(a string, b string) bool {
	if sameSeverity(a, b) {
		return false
	}

	if a == TypeResult {
		return true
	}

	if a == TypeError {
		return b != TypeResult
	}

	// a is either log or event
	return !(b == TypeResult || b == TypeError)
}
