package data

import (
	"time"
)

type RetryPolicy struct {
	Count int32  `json:"count,omitempty"`
	Until string `json:"until,omitempty" expr:"expression"`
}

type StepData struct {
	Status        *StepStatus `json:"s,omitempty"`
	StartedAt     *time.Time  `json:"S,omitempty"`
	Condition     string      `json:"c,omitempty"`
	Parents       []string    `json:"p,omitempty"`
	Timeout       string      `json:"t,omitempty"`
	PausedOnStart bool        `json:"P,omitempty"`
	Retry         RetryPolicy `json:"r,omitempty"`
	Result        string      `json:"R,omitempty"`
}

func (s *StepData) SetCondition(expression string) *StepData {
	s.Condition = expression
	return s
}

func (s *StepData) SetParents(parents []string) *StepData {
	s.Parents = parents
	return s
}

func (s *StepData) SetPausedOnStart(pause bool) *StepData {
	s.PausedOnStart = pause
	return s
}

func (s *StepData) SetTimeout(timeout string) *StepData {
	s.Timeout = timeout
	return s
}

func (s *StepData) SetResult(expression string) *StepData {
	s.Result = expression
	return s
}

func (s *StepData) SetRetryPolicy(policy RetryPolicy) *StepData {
	s.Retry = policy
	return s
}

func (s *StepData) SetStatus(status StepStatus) *StepData {
	s.Status = &status
	return s
}
