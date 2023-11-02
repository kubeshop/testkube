package events

import (
	"bytes"
	"regexp"
	"time"

	"encoding/json"

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
type Type string

const (
	// v1 - old log format based on shell output of executors {"line":"...", "time":"..."}
	LogVersionV1 LogVersion = "v1"
	// v2 - raw binary format, timestamps are based on Kubernetes logs, line is raw log line
	LogVersionV2 LogVersion = "v2"

	// TypeTestPod - logs from test pod (all containers)
	TypeTestPod Type = "test-execution-pod"
	// TypeSchduler - logs from scheduler pod
	TypeSchduler Type = "test-scheduler"
	// TypeOperator - logs from operator pod
	TypeOperator Type = "operator"
)

type LogOutputV1 struct {
	Result *testkube.ExecutionResult
}

type LogChunk struct {
	Time     time.Time         `json:"ts,omitempty"`
	Content  string            `json:"content,omitempty"`
	Type     string            `json:"type,omitempty"`
	Source   string            `json:"source,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Error    error             `json:"error,omitempty"`
	Version  LogVersion        `json:"version,omitempty"`

	// Old output - for backwards compatibility - will be removed
	V1 *LogOutputV1 `json:"v1,omitempty"`
}

func NewLogChunk(ts time.Time, content []byte) LogChunk {
	return LogChunk{
		Time:     ts,
		Content:  string(content),
		Metadata: map[string]string{},
	}
}

// log line/chunk data
func (c *LogChunk) WithMetadataEntry(key, value string) *LogChunk {
	if c.Metadata == nil {
		c.Metadata = map[string]string{}
	}
	c.Metadata[key] = value
	return c
}

func (c *LogChunk) WithVersion(version LogVersion) *LogChunk {
	c.Version = version
	return c
}

func (c *LogChunk) WithV1Result(result *testkube.ExecutionResult) *LogChunk {
	c.V1.Result = result
	return c
}

// TODO handle errrors
func (c LogChunk) Encode() []byte {
	b, _ := json.Marshal(c)
	return b
}

var rsRegexp = regexp.MustCompile("^[0-9]{4}-[0-9]{2}-[0-9]{2}T.*")

// NewLogChunkFromBytes creates new LogChunk from bytes it's aware of new and old log formats
// default log format will be based on raw bytes with timestamp on the beginning
func NewLogChunkFromBytes(b []byte) LogChunk {

	// detect timestamp - new logs have timestamp
	var (
		hasTimestamp bool
		ts           time.Time
		content      []byte
		err          error
	)

	if rsRegexp.Match(b) {
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
			return LogChunk{
				Time:    ts,
				Content: o.Content,
				Error:   err,
				Version: LogVersionV1,
			}
		}

		// pass parsed results for v1
		// for new executor it'll be omitted in logs (as looks like we're not using it already)
		if o.Type_ == output.TypeResult {
			return LogChunk{
				Time:    ts,
				Content: o.Content,
				Version: LogVersionV1,
				V1: &LogOutputV1{
					Result: o.Result,
				},
			}
		}

		return LogChunk{
			Time:    ts,
			Content: o.Content,
			Version: LogVersionV1,
		}
	}
	// END DEPRECATED

	// new non-JSON format (just raw lines will be logged)
	return LogChunk{
		Time:    ts,
		Content: string(b),
		Version: LogVersionV2,
	}
}
