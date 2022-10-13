package triggers

import (
	"fmt"
	"time"

	testtriggersv1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
)

type statusKey string

func newStatusKey(namespace, name string) statusKey {
	return statusKey(fmt.Sprintf("%s/%s", namespace, name))
}

type triggerStatus struct {
	testTrigger           *testtriggersv1.TestTrigger
	lastExecutionStarted  *time.Time
	lastExecutionFinished *time.Time
	testExecutionIDs      []string
	testSuiteExecutionIDs []string
}

func newTriggerStatus(testTrigger *testtriggersv1.TestTrigger) *triggerStatus {
	return &triggerStatus{testTrigger: testTrigger}
}

func (s *triggerStatus) hasActiveTests() bool {
	return len(s.testExecutionIDs) > 0 || len(s.testSuiteExecutionIDs) > 0
}

func (s *triggerStatus) start() {
	now := time.Now()
	s.lastExecutionStarted = &now
	s.lastExecutionFinished = nil
}

func (s *triggerStatus) addExecutionID(id string) {
	s.testExecutionIDs = append(s.testExecutionIDs, id)
}

func (s *triggerStatus) removeExecutionID(targetID string) {
	for i, id := range s.testExecutionIDs {
		if id == targetID {
			s.testExecutionIDs = append(s.testExecutionIDs[:i], s.testExecutionIDs[i+1:]...)
		}
	}
}

func (s *triggerStatus) addTestSuiteExecutionID(id string) {
	s.testSuiteExecutionIDs = append(s.testSuiteExecutionIDs, id)
}

func (s *triggerStatus) removeTestSuiteExecutionID(targetID string) {
	for i, id := range s.testSuiteExecutionIDs {
		if id == targetID {
			s.testSuiteExecutionIDs = append(s.testSuiteExecutionIDs[:i], s.testSuiteExecutionIDs[i+1:]...)
		}
	}
}

func (s *triggerStatus) done() {
	now := time.Now()
	s.lastExecutionFinished = &now
	s.testExecutionIDs = nil
	s.testSuiteExecutionIDs = nil
}

func (s *Service) getStatusForTrigger(t *testtriggersv1.TestTrigger) *triggerStatus {
	key := newStatusKey(t.Namespace, t.Name)
	return s.triggerStatus[key]
}
