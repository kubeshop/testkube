package triggers

import (
	"fmt"
	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"time"
)

type StatusKey string

func NewStatusKey(namespace, name string) StatusKey {
	return StatusKey(fmt.Sprintf("%s/%s", namespace, name))
}

type TriggerStatus struct {
	ActiveTests           bool
	LastExecutionStarted  *time.Time
	LastExecutionFinished *time.Time
}

func (s *TriggerStatus) TestsStarted() {
	s.ActiveTests = true
	now := time.Now()
	s.LastExecutionStarted = &now
	s.LastExecutionFinished = nil
}

func (s *TriggerStatus) TestsStopped() {
	s.ActiveTests = false
	now := time.Now()
	s.LastExecutionFinished = &now
}

func NewTriggerStatus() *TriggerStatus {
	return &TriggerStatus{}
}

func (s *Service) getStatusForTrigger(t *testtriggersv1.TestTrigger) *TriggerStatus {
	key := NewStatusKey(t.Namespace, t.Name)
	return s.status[key]
}
