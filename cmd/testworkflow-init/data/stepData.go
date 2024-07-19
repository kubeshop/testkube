package data

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

type RetryPolicy struct {
	Count int32  `json:"count,omitempty"`
	Until string `json:"until,omitempty" expr:"expression"`
}

type StepData struct {
	Ref           string         `json:"_,omitempty"`
	ExitCode      uint8          `json:"e,omitempty"`
	Status        *StepStatus    `json:"s,omitempty"`
	StartedAt     *time.Time     `json:"S,omitempty"`
	Condition     string         `json:"c,omitempty"`
	Parents       []string       `json:"p,omitempty"`
	Timeout       *time.Duration `json:"t,omitempty"`
	PausedOnStart bool           `json:"P,omitempty"`
	Retry         RetryPolicy    `json:"r,omitempty"`
	Result        string         `json:"R,omitempty"`
	Iteration     int32          `json:"i,omitempty"`

	// Pausing
	PausedNs    int64      `json:"n,omitempty"`
	PausedStart *time.Time `json:"N,omitempty"`
	paused      bool
	pauseMu     sync.Mutex
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

func (s *StepData) SetExitCode(exitCode uint8) *StepData {
	s.ExitCode = exitCode
	return s
}

func (s *StepData) SetCondition(expression string) *StepData {
	s.Condition = expression
	return s
}

func (s *StepData) SetParents(parents []string) *StepData {
	parents = slices.Clone(parents)
	slices.Reverse(parents)
	s.Parents = parents
	return s
}

func (s *StepData) SetPausedOnStart(pause bool) *StepData {
	s.PausedOnStart = pause
	return s
}

func (s *StepData) SetTimeout(timeout string) *StepData {
	if timeout == "" {
		s.Timeout = nil
	}
	duration, err := time.ParseDuration(timeout)
	if err != nil {
		panic(fmt.Sprintf("invalid timeout duration: %s: %s", timeout, err.Error()))
	}
	s.Timeout = &duration
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

func (s *StepData) RegisterPauseStart(ts time.Time) bool {
	s.pauseMu.Lock()
	defer s.pauseMu.Unlock()

	if s.paused {
		return false
	}
	s.paused = true
	start := ts
	s.PausedStart = &start
	return true
}

func (s *StepData) RegisterPauseEnd(ts time.Time) bool {
	s.pauseMu.Lock()
	defer s.pauseMu.Unlock()

	if !s.paused {
		return false
	}
	s.paused = false
	took := ts.Sub(*s.PausedStart)
	s.PausedStart = nil
	s.PausedNs += took.Nanoseconds()
	return true
}

func (s *StepData) Took(ts time.Time) time.Duration {
	s.pauseMu.Lock()
	defer s.pauseMu.Unlock()

	if s.StartedAt == nil {
		return 0
	}

	if s.PausedStart != nil {
		return ts.Sub(*s.PausedStart) - time.Duration(s.PausedNs)
	}
	return ts.Sub(*s.StartedAt) - time.Duration(s.PausedNs)
}

func (s *StepData) TimeLeft(ts time.Time) *time.Duration {
	if s.Timeout == nil || s.StartedAt == nil {
		return nil
	}
	left := *s.Timeout - s.Took(ts)
	return &left
}
