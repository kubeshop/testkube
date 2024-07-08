package data

import (
	"time"
)

type RetryPolicy struct {
	Count int32  `json:"count,omitempty"`
	Until string `json:"until,omitempty" expr:"expression"`
}

type StepData struct {
	Status    *StepStatus `json:"s,omitempty"`
	StartedAt *time.Time  `json:"S,omitempty"`
	Condition string      `json:"c,omitempty"`
	Parents   []string    `json:"p,omitempty"`
	Timeout   string      `json:"t,omitempty"`
	Paused    bool        `json:"P,omitempty"`
	Retry     RetryPolicy `json:"r,omitempty"`
	Result    string      `json:"R,omitempty"`
}
