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
	ExecutionIDs          []string
	TestSuiteExecutionIDs []string
}

func (s *TriggerStatus) Start() {
	s.ActiveTests = true
	now := time.Now()
	s.LastExecutionStarted = &now
	s.LastExecutionFinished = nil
}

func (s *TriggerStatus) AddExecutionID(id string) {
	s.ExecutionIDs = append(s.ExecutionIDs, id)
}

func (s *TriggerStatus) RemoveExecutionID(targetID string) {
	for i, id := range s.ExecutionIDs {
		if id == targetID {
			s.ExecutionIDs = append(s.ExecutionIDs[:i], s.ExecutionIDs[i+1:]...)
		}
	}
}

func (s *TriggerStatus) AddTestSuiteExecutionID(id string) {
	s.ExecutionIDs = append(s.TestSuiteExecutionIDs, id)
}

func (s *TriggerStatus) RemoveTestSuiteExecutionID(targetID string) {
	for i, id := range s.TestSuiteExecutionIDs {
		if id == targetID {
			s.ExecutionIDs = append(s.TestSuiteExecutionIDs[:i], s.TestSuiteExecutionIDs[i+1:]...)
		}
	}
}

func (s *TriggerStatus) Finish() {
	s.ActiveTests = false
	now := time.Now()
	s.LastExecutionFinished = &now
	s.ExecutionIDs = nil
	s.TestSuiteExecutionIDs = nil
}

func NewTriggerStatus() *TriggerStatus {
	return &TriggerStatus{}
}

func (s *Service) getStatusForTrigger(t *testtriggersv1.TestTrigger) *TriggerStatus {
	key := NewStatusKey(t.Namespace, t.Name)
	return s.triggerStatus[key]
}
