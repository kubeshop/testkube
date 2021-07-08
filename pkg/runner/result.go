package runner

import "time"

type Result struct {
	ID            string
	Name          string
	ScriptType    string
	Status        string
	StartTime     time.Time
	EndTime       time.Time
	ExecutionTime time.Duration
	Output        interface{}
	OutputType    string
}
