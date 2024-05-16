package triggers

import (
	"fmt"
	"sync"
	"time"

	testtriggersv1 "github.com/kubeshop/testkube-operator/api/testtriggers/v1"
)

type statusKey string

func newStatusKey(namespace, name string) statusKey {
	return statusKey(fmt.Sprintf("%s/%s", namespace, name))
}

type triggerStatus struct {
	testTrigger              *testtriggersv1.TestTrigger
	lastExecutionStarted     *time.Time
	lastExecutionFinished    *time.Time
	testExecutionIDs         []string
	testSuiteExecutionIDs    []string
	testWorkflowExecutionIDs []string
	sync.RWMutex
}

func newTriggerStatus(testTrigger *testtriggersv1.TestTrigger) *triggerStatus {
	return &triggerStatus{testTrigger: testTrigger}
}

func (s *triggerStatus) hasActiveTests() bool {
	defer s.RUnlock()

	s.RLock()
	return len(s.testExecutionIDs) > 0 || len(s.testSuiteExecutionIDs) > 0 || len(s.testWorkflowExecutionIDs) > 0
}

func (s *triggerStatus) getExecutionIDs() []string {
	defer s.RUnlock()

	s.RLock()
	executionIDs := make([]string, len(s.testExecutionIDs))
	copy(executionIDs, s.testExecutionIDs)

	return executionIDs
}

func (s *triggerStatus) getTestSuiteExecutionIDs() []string {
	defer s.RUnlock()

	s.RLock()
	testSuiteExecutionIDs := make([]string, len(s.testSuiteExecutionIDs))
	copy(testSuiteExecutionIDs, s.testSuiteExecutionIDs)

	return testSuiteExecutionIDs
}

func (s *triggerStatus) getTestWorkflowExecutionIDs() []string {
	defer s.RUnlock()

	s.RLock()
	testWorkflowExecutionIDs := make([]string, len(s.testWorkflowExecutionIDs))
	copy(testWorkflowExecutionIDs, s.testWorkflowExecutionIDs)

	return testWorkflowExecutionIDs
}

func (s *triggerStatus) start() {
	defer s.Unlock()

	s.Lock()
	now := time.Now()
	s.lastExecutionStarted = &now
	s.lastExecutionFinished = nil
}

func (s *triggerStatus) addExecutionID(id string) {
	defer s.Unlock()

	s.Lock()
	s.testExecutionIDs = append(s.testExecutionIDs, id)
}

func (s *triggerStatus) removeExecutionID(targetID string) {
	defer s.Unlock()

	s.Lock()
	for i, id := range s.testExecutionIDs {
		if id == targetID {
			s.testExecutionIDs = append(s.testExecutionIDs[:i], s.testExecutionIDs[i+1:]...)
		}
	}
}

func (s *triggerStatus) addTestSuiteExecutionID(id string) {
	defer s.Unlock()

	s.Lock()
	s.testSuiteExecutionIDs = append(s.testSuiteExecutionIDs, id)
}

func (s *triggerStatus) removeTestSuiteExecutionID(targetID string) {
	defer s.Unlock()

	s.Lock()
	for i, id := range s.testSuiteExecutionIDs {
		if id == targetID {
			s.testSuiteExecutionIDs = append(s.testSuiteExecutionIDs[:i], s.testSuiteExecutionIDs[i+1:]...)
		}
	}
}

func (s *triggerStatus) addTestWorkflowExecutionID(id string) {
	defer s.Unlock()

	s.Lock()
	s.testWorkflowExecutionIDs = append(s.testWorkflowExecutionIDs, id)
}

func (s *triggerStatus) removeTestWorkflowExecutionID(targetID string) {
	defer s.Unlock()

	s.Lock()
	for i, id := range s.testWorkflowExecutionIDs {
		if id == targetID {
			s.testWorkflowExecutionIDs = append(s.testWorkflowExecutionIDs[:i], s.testWorkflowExecutionIDs[i+1:]...)
		}
	}
}

func (s *triggerStatus) done() {
	defer s.Unlock()

	s.Lock()
	now := time.Now()
	s.lastExecutionFinished = &now
}

func (s *Service) getStatusForTrigger(t *testtriggersv1.TestTrigger) *triggerStatus {
	key := newStatusKey(t.Namespace, t.Name)
	return s.triggerStatus[key]
}
