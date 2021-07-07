package runner

import "time"

type Result struct {
	StartTime     time.Time
	EndTime       time.Time
	ExecutionTime time.Duration
	Output        interface{}
}
