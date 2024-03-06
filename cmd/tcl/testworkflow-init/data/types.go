// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package data

import (
	"strings"
	"time"
)

type TestWorkflowStatus string

const (
	TestWorkflowStatusPassed  TestWorkflowStatus = ""
	TestWorkflowStatusFailed  TestWorkflowStatus = "failed"
	TestWorkflowStatusAborted TestWorkflowStatus = "aborted"
)

type StepStatus string

const (
	StepStatusPassed  StepStatus = ""
	StepStatusTimeout StepStatus = "timeout"
	StepStatusFailed  StepStatus = "failed"
	StepStatusAborted StepStatus = "aborted"
	StepStatusSkipped StepStatus = "skipped"
)

type Rule struct {
	Expr string
	Refs []string
}

type Timeout struct {
	Ref      string
	Duration string
}

type StepInfo struct {
	Ref       string     `json:"ref"`
	Status    StepStatus `json:"status"`
	HasStatus bool       `json:"hasStatus"`
	StartTime time.Time  `json:"startTime"`
	TimeoutAt time.Time  `json:"timeoutAt"`
	Iteration uint64     `json:"iteration"`
}

func (s *StepInfo) Start(t time.Time) {
	if s.StartTime.IsZero() {
		s.StartTime = t
		s.Iteration = 1
		PrintHint(s.Ref, "start")
	}
}

func (s *StepInfo) Next() {
	if s.StartTime.IsZero() {
		s.Start(time.Now())
	} else {
		s.Iteration++
		PrintHintDetails(s.Ref, "iteration", s.Iteration)
	}
}

func (s *StepInfo) Skip(t time.Time) {
	if s.Status != StepStatusSkipped {
		s.StartTime = t
		s.Iteration = 0
		s.SetStatus(StepStatusSkipped)
	}
}

func (s *StepInfo) SetTimeoutDuration(t time.Time, duration string) error {
	if !s.TimeoutAt.IsZero() {
		return nil
	}
	s.Start(t)
	v, err := Template(duration)
	if err != nil {
		return err
	}
	d, err := time.ParseDuration(strings.ReplaceAll(v, " ", ""))
	if err != nil {
		return err
	}
	s.TimeoutAt = s.StartTime.Add(d)
	return nil
}

func (s *StepInfo) SetStatus(status StepStatus) {
	if !s.HasStatus || s.Status == StepStatusPassed {
		s.Status = status
		s.HasStatus = true
		if status == StepStatusPassed {
			PrintHintDetails(s.Ref, "status", "passed")
		} else {
			PrintHintDetails(s.Ref, "status", status)
		}
	}
}
