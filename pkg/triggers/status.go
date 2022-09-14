package triggers

import (
	"fmt"
	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"time"
)

type statusKey string

func newStatusKey(namespace, name string) statusKey {
	return statusKey(fmt.Sprintf("%s/%s", namespace, name))
}

type triggerStatus struct {
	ActiveTests           bool
	LastExecutionStarted  *time.Time
	LastExecutionFinished *time.Time
	ExecutionIDs          []string
	TestSuiteExecutionIDs []string
}

func newTriggerStatus() *triggerStatus {
	return &triggerStatus{}
}

func (s *triggerStatus) start() {
	s.ActiveTests = true
	now := time.Now()
	s.LastExecutionStarted = &now
	s.LastExecutionFinished = nil
}

func (s *triggerStatus) addExecutionID(id string) {
	s.ExecutionIDs = append(s.ExecutionIDs, id)
}

func (s *triggerStatus) removeExecutionID(targetID string) {
	for i, id := range s.ExecutionIDs {
		if id == targetID {
			s.ExecutionIDs = append(s.ExecutionIDs[:i], s.ExecutionIDs[i+1:]...)
		}
	}
}

func (s *triggerStatus) addTestSuiteExecutionID(id string) {
	s.ExecutionIDs = append(s.TestSuiteExecutionIDs, id)
}

func (s *triggerStatus) removeTestSuiteExecutionID(targetID string) {
	for i, id := range s.TestSuiteExecutionIDs {
		if id == targetID {
			s.ExecutionIDs = append(s.TestSuiteExecutionIDs[:i], s.TestSuiteExecutionIDs[i+1:]...)
		}
	}
}

func (s *triggerStatus) finish() {
	s.ActiveTests = false
	now := time.Now()
	s.LastExecutionFinished = &now
	s.ExecutionIDs = nil
	s.TestSuiteExecutionIDs = nil
}

func (s *Service) getStatusForTrigger(t *testtriggersv1.TestTrigger) *triggerStatus {
	key := newStatusKey(t.Namespace, t.Name)
	return s.triggerStatus[key]
}
