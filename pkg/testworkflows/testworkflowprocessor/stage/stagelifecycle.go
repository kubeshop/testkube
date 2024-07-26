package stage

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
	Paused() bool
	RetryPolicy() testworkflowsv1.RetryPolicy
	Timeout() string

	SetNegative(negative bool) StageLifecycle
	SetOptional(optional bool) StageLifecycle
	SetCondition(expr string) StageLifecycle
	SetPaused(paused bool) StageLifecycle
	AppendConditions(expr ...string) StageLifecycle
	SetRetryPolicy(policy *testworkflowsv1.RetryPolicy) StageLifecycle
	SetTimeout(tpl string) StageLifecycle
}

type stageLifecycle struct {
	negative  bool
	optional  bool
	paused    bool
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

func (s *stageLifecycle) Paused() bool {
	return s.paused
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

func (s *stageLifecycle) SetPaused(paused bool) StageLifecycle {
	s.paused = paused
	return s
}
