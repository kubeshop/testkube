package output

import (
	"encoding/json"
	"time"
)

func GetLogEntry(b []byte) (out Output) {
	if len(b) == 0 {
		return Output{
			Type_:   TypeUnknown,
			Content: "",
			Time:    time.Now(),
		}
	}

	// not json
	if b[0] != byte('{') {
		return Output{
			Type_:   TypeLogLine,
			Content: string(b),
			Time:    time.Now(),
		}
	}

	err := json.Unmarshal(b, &out)
	if err != nil {
		return Output{
			Type_:   TypeLogLine,
			Content: string(b),
			Time:    time.Now(),
		}
	}

	if out.Type_ == "" {
		out.Type_ = TypeUnknown
	}

	// fallback to raw content if no content in the parsed log
	if out.Content == "" {
		out.Content = string(b)
	}

	return out
}
