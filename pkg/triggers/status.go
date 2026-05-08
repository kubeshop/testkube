package triggers

import (
	"fmt"
	"sync"
	"time"

	testtriggersv1 "github.com/kubeshop/testkube/api/testtriggers/v1"
)

type statusKey string

func newStatusKey(source, namespace, name string) statusKey {
	return statusKey(fmt.Sprintf("%s:%s/%s", source, namespace, name))
}

type triggerStatus struct {
	trigger                  *internalTrigger
	lastExecutionStarted     *time.Time
	lastExecutionFinished    *time.Time
	testExecutionIDs         []string
	testSuiteExecutionIDs    []string
	testWorkflowExecutionIDs []string
	sync.RWMutex
}

func newTriggerStatusFromV1(t *testtriggersv1.TestTrigger) *triggerStatus {
	return &triggerStatus{trigger: convertV1ToInternal(t)}
}

func (s *triggerStatus) hasActiveTests() bool {
	defer s.RUnlock()

	s.RLock()
	return len(s.testExecutionIDs) > 0 || len(s.testSuiteExecutionIDs) > 0 || len(s.testWorkflowExecutionIDs) > 0
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

func (s *Service) getStatusForTrigger(t *internalTrigger) *triggerStatus {
	key := newStatusKey(t.Source, t.Namespace, t.Name)
	s.triggerStatusMu.RLock()
	defer s.triggerStatusMu.RUnlock()
	return s.triggerStatus[key]
}
