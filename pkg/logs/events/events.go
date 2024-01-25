package events

import (
	"bytes"
	"regexp"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

// Generic event like log-start log-end
type Trigger struct {
	Id       string            `json:"id,omitempty"`
	Type     string            `json:"type,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type LogVersion string

const (
	// v1 - old log format based on shell output of executors {"line":"...", "time":"..."}
	LogVersionV1 LogVersion = "v1"
	// v2 - raw binary format, timestamps are based on Kubernetes logs, line is raw log line
	LogVersionV2 LogVersion = "v2"

	JobPodLogSource = "job-pod"
)

type LogResponse struct {
	Log   Log
	Error error
}

type Log struct {
	Time     time.Time         `json:"ts,omitempty"`
	Content  string            `json:"content,omitempty"`
	Type     string            `json:"type,omitempty"`
	Source   string            `json:"source,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Error    bool              `json:"error,omitempty"`
	Version  LogVersion        `json:"version,omitempty"`

	// Old output - for backwards compatibility - will be removed for non-structured logs
	V1 *LogOutputV1 `json:"v1,omitempty"`
}
type LogOutputV1 struct {
	Result *testkube.ExecutionResult
}

func NewErrorLog(err error) *Log {
	var msg string
	if err != nil {
		msg = err.Error()
	}
	return &Log{
		Error:   true,
		Content: msg,
	}
}

func NewLog(content string) *Log {
	return &Log{
		Time:     time.Now(),
		Content:  string(content),
		Metadata: map[string]string{},
	}
}

func NewLogTs(ts time.Time, content []byte) Log {
	return Log{
		Time:     ts,
		Content:  string(content),
		Metadata: map[string]string{},
	}
}

func (l *Log) WithMetadataEntry(key, value string) *Log {
	if l.Metadata == nil {
		l.Metadata = map[string]string{}
	}
	l.Metadata[key] = value
	return l
}

func (l *Log) WithType(t string) *Log {
	l.Type = t
	return l
}

func (l *Log) WithSource(s string) *Log {
	l.Source = s
	return l
}

func (l *Log) WithVersion(version LogVersion) *Log {
	l.Version = version
	return l
}

func (l *Log) WithV1Result(result *testkube.ExecutionResult) *Log {
	l.V1.Result = result
	return l
}

var timestampRegexp = regexp.MustCompile("^[0-9]{4}-[0-9]{2}-[0-9]{2}T.*")

// NewLogFromBytes creates new LogResponse from bytes it's aware of new and old log formats
// default log format will be based on raw bytes with timestamp on the beginning
func NewLogFromBytes(b []byte) *Log {

	// detect timestamp - new logs have timestamp
	var (
		hasTimestamp bool
		ts           time.Time
		content      []byte
		err          error
	)

	if timestampRegexp.Match(b) {
		hasTimestamp = true
	}

	// if there is output with timestamp
	if hasTimestamp {
		s := bytes.SplitN(b, []byte(" "), 2)
		ts, err = time.Parse(time.RFC3339Nano, string(s[0]))
		// fallback to now in case of error
		if err != nil {
			ts = time.Now()
		}

		content = s[1]
	} else {
		ts = time.Now()
		content = b
	}

	// DEPRECATED - old log format
	// detect JSON and try to parse old log structure

	// We need .Content if available
	// .Time - is not needed at all - timestamp will be get from Kubernetes logs
	// One thing which need to be handled is result
	// .Result

	if bytes.HasPrefix(content, []byte("{")) {
		o, err := output.GetLogEntry(content)
		if err != nil {
			// try to read in case of some lines which we couldn't parse
			// sometimes we're not able to control all stdout messages from libs
			return &Log{
				Time:    ts,
				Content: err.Error(),
				Type:    o.Type_,
				Error:   true,
				Version: LogVersionV1,
				Source:  JobPodLogSource,
			}
		}

		// pass parsed results for v1
		// for new executor it'll be omitted in logs (as looks like we're not using it already)
		if o.Type_ == output.TypeResult {
			return &Log{
				Time:    ts,
				Content: o.Content,
				Version: LogVersionV1,
				V1: &LogOutputV1{
					Result: o.Result,
				},
			}
		}

		return &Log{
			Time:    ts,
			Content: o.Content,
			Version: LogVersionV1,
		}
	}
	// END DEPRECATED

	// new non-JSON format (just raw lines will be logged)
	return &Log{
		Time:    ts,
		Content: string(b),
		Version: LogVersionV2,
		Source:  JobPodLogSource,
	}
}
