package data

import (
	"errors"
	"time"
)

type RetryPolicy struct {
	Count int32  `json:"count,omitempty"`
	Until string `json:"until,omitempty" expr:"expression"`
}

type StepData struct {
	Ref           string      `json:"_,omitempty"`
	Status        *StepStatus `json:"s,omitempty"`
	StartedAt     *time.Time  `json:"S,omitempty"`
	Condition     string      `json:"c,omitempty"`
	Parents       []string    `json:"p,omitempty"`
	Timeout       string      `json:"t,omitempty"`
	PausedOnStart bool        `json:"P,omitempty"`
	Retry         RetryPolicy `json:"r,omitempty"`
	Result        string      `json:"R,omitempty"`
}

func (s *StepData) IsFinished() bool {
	return s.Status != nil
}

func (s *StepData) IsStarted() bool {
	return s.StartedAt != nil
}

func (s *StepData) ResolveCondition() (bool, error) {
	if s.Condition == "" {
		return false, errors.New("missing condition expression")
	}
	expr, err := Expression(s.Condition, RefSuccessMachine)
	if err != nil {
		return false, err
	}
	return expr.Static().BoolValue()
}

func (s *StepData) ResolveResult() (StepStatus, error) {
	if s.Result == "" {
		return StepStatusAborted, errors.New("missing result expression")
	}
	expr, err := Expression(s.Result, RefSuccessMachine)
	if err != nil {
		return StepStatusAborted, err
	}
	success, err := expr.Static().BoolValue()
	if err != nil {
		return StepStatusAborted, err
	}
	if success {
		return StepStatusPassed, nil
	}
	return StepStatusFailed, nil
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
