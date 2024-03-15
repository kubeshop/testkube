// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package testworkflowprocessor

import (
	"strings"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
)

type Condition struct {
	Refs      []string
	Condition string `expr:"expression"`
}

type StageLifecycle interface {
	Negative() bool
	Optional() bool
	Condition() string
	RetryPolicy() testworkflowsv1.RetryPolicy
	Timeout() string

	SetNegative(negative bool) StageLifecycle
	SetOptional(optional bool) StageLifecycle
	SetCondition(expr string) StageLifecycle
	AppendConditions(expr ...string) StageLifecycle
	SetRetryPolicy(policy *testworkflowsv1.RetryPolicy) StageLifecycle
	SetTimeout(tpl string) StageLifecycle
}

type stageLifecycle struct {
	negative  bool
	optional  bool
	condition string
	retry     testworkflowsv1.RetryPolicy
	timeout   string
}

func NewStageLifecycle() StageLifecycle {
	return &stageLifecycle{}
}

func (s *stageLifecycle) Negative() bool {
	return s.negative
}

func (s *stageLifecycle) Optional() bool {
	return s.optional
}

func (s *stageLifecycle) Condition() string {
	return s.condition
}

func (s *stageLifecycle) RetryPolicy() testworkflowsv1.RetryPolicy {
	if s.retry.Count < 1 {
		s.retry.Count = 0
	}
	return s.retry
}

func (s *stageLifecycle) Timeout() string {
	return s.timeout
}

func (s *stageLifecycle) SetNegative(negative bool) StageLifecycle {
	s.negative = negative
	return s
}

func (s *stageLifecycle) SetOptional(optional bool) StageLifecycle {
	s.optional = optional
	return s
}

func (s *stageLifecycle) SetCondition(expr string) StageLifecycle {
	s.condition = expr
	return s
}

func (s *stageLifecycle) AppendConditions(expr ...string) StageLifecycle {
	expr = append(expr, s.condition)
	cond := []string(nil)
	seen := map[string]bool{} // Assume pure accessors in condition, and preserve only unique
	for _, e := range expr {
		if e != "" && !seen[e] {
			seen[e] = true
			cond = append(cond, e)
		}
	}

	s.condition = strings.Join(cond, "&&")

	return s
}

func (s *stageLifecycle) SetRetryPolicy(policy *testworkflowsv1.RetryPolicy) StageLifecycle {
	if policy == nil {
		policy = &testworkflowsv1.RetryPolicy{}
	}
	s.retry = *policy
	return s
}

func (s *stageLifecycle) SetTimeout(tpl string) StageLifecycle {
	s.timeout = tpl
	return s
}
