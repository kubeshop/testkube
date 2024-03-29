package events

import (
	"bytes"
	"encoding/json"
	"regexp"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

type LogVersion string

const (
	// v1 - old log format based on shell output of executors {"line":"...", "time":"..."}
	LogVersionV1 LogVersion = "v1"
	// v2 - raw binary format, timestamps are based on Kubernetes logs, line is raw log line
	LogVersionV2 LogVersion = "v2"

	SourceJobPod            = "job-pod"
	SourceScheduler         = "test-scheduler"
	SourceContainerExecutor = "container-executor"
	SourceJobExecutor       = "job-executor"
	SourceLogsProxy         = "logs-proxy"
)

// check if trigger implements model generic event type
var _ testkube.Trigger = Trigger{}

// NewTrigger returns Trigger instance
func NewTrigger(id string) Trigger {
	return Trigger{ResourceId: id}
}

// Generic event like log-start log-end with resource id
type Trigger struct {
	ResourceId string `json:"resourceId,omitempty"`
}

// GetResourceId implements testkube.Trigger interface
func (t Trigger) GetResourceId() string {
	return t.ResourceId
}

type LogResponse struct {
	Log   Log
	Error error
}

type Log testkube.LogV2

func NewFinishLog() *Log {
	return &Log{
		Time:    time.Now(),
		Content: "processing logs finished",
		Type_:   "finish",
		Source:  "log-server",
	}
}

func IsFinished(log *Log) bool {
	return log.Type_ == "finish"
}

func NewErrorLog(err error) *Log {
	var msg string
	if err != nil {
		msg = err.Error()
	}
	return &Log{
		Time:    time.Now(),
		Error_:  true,
		Content: msg,
	}
}

func NewLog(content ...string) *Log {
	log := &Log{
		Time:     time.Now(),
		Metadata: map[string]string{},
	}

	if len(content) > 0 {
		log.WithContent(content[0])
	}

	return log
}

func (l *Log) WithContent(s string) *Log {
	l.Content = s
	return l
}

func (l *Log) WithError(err error) *Log {
	l.Error_ = true

	if err != nil {
		l.Content = err.Error()
	}

	return l
}

func (l *Log) WithMetadataEntry(key, value string) *Log {
	if l.Metadata == nil {
		l.Metadata = map[string]string{}
	}
	l.Metadata[key] = value
	return l
}

func (l *Log) WithType(t string) *Log {
	l.Type_ = t
	return l
}

func (l *Log) WithSource(s string) *Log {
	l.Source = s
	return l
}

func (l *Log) WithVersion(version LogVersion) *Log {
	l.Version = string(version)
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
			return newErrorLog(err, content)
		}

		// pass parsed results for v1
		// for new executor it'll be omitted in logs (as looks like we're not using it already)
		if o.Type_ == output.TypeResult {
			return &Log{
				Time:    ts,
				Content: o.Content,
				Version: string(LogVersionV1),
				V1: &testkube.LogV1{
					Result: o.Result,
				},
			}
		}

		return &Log{
			Time:    ts,
			Content: o.Content,
			Version: string(LogVersionV1),
		}
	}
	// END DEPRECATED

	// new non-JSON format (just raw lines will be logged)
	return &Log{
		Time:    ts,
		Content: string(content),
		Version: string(LogVersionV2),
	}
}

// ReadLogLine tries to read possible log lines from any source
// - logv2 - JSON
// - logv1 - old log format JSON - DEPRECATED
// - possible errors or raw log lines
func ReadLogLine(b []byte) *Log {
	logsV1Prefix := []byte("{\"id\"")
	logsV2Prefix := []byte("{")

	switch true {
	case bytes.HasPrefix(b, logsV1Prefix):
		o, err := output.GetLogEntry(b)
		if err != nil {
			return newErrorLog(err, b)
		}
		return mapLogV1toV2(o)

	case bytes.HasPrefix(b, logsV2Prefix):
		var o Log
		err := json.Unmarshal(b, &o)
		if err != nil {
			return newErrorLog(err, b)
		}
		return &o
	}

	return &Log{
		Content: string(b),
	}
}

func newErrorLog(err error, b []byte) *Log {
	return &Log{
		Content:  string(b),
		Error_:   true,
		Version:  string(LogVersionV1),
		Metadata: map[string]string{"error": err.Error()},
	}

}

func mapLogV1toV2(o output.Output) *Log {
	// pass parsed results for v1
	// for new executor it'll be omitted in logs (as looks like we're not using it already)
	if o.Type_ == output.TypeResult {
		return &Log{
			Time:    o.Time,
			Content: o.Content,
			Version: string(LogVersionV1),
			V1: &testkube.LogV1{
				Result: o.Result,
			},
		}
	}

	return &Log{
		Time:    o.Time,
		Content: o.Content,
		Version: string(LogVersionV1),
	}

}
