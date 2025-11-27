package runner

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type eventRecorder struct {
	mu     sync.Mutex
	events []testkube.Event
}

func (r *eventRecorder) Notify(event testkube.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *eventRecorder) drain() []testkube.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]testkube.Event, len(r.events))
	copy(cp, r.events)
	return cp
}

func TestNotifyWorkflowCompleted_EmitsEventsForStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status testkube.TestWorkflowStatus
		expect *testkube.EventType
	}{
		{name: "passed", status: testkube.PASSED_TestWorkflowStatus, expect: testkube.EventEndTestWorkflowSuccess},
		{name: "aborted", status: testkube.ABORTED_TestWorkflowStatus, expect: testkube.EventEndTestWorkflowAborted},
		{name: "canceled", status: testkube.CANCELED_TestWorkflowStatus, expect: testkube.EventEndTestWorkflowCanceled},
		{name: "failed", status: testkube.FAILED_TestWorkflowStatus, expect: testkube.EventEndTestWorkflowFailed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &eventRecorder{}
			exec := &testkube.TestWorkflowExecution{
				Id: "exec",
				Result: &testkube.TestWorkflowResult{
					Status: &tc.status,
				},
			}
			notifyWorkflowCompleted(rec, exec)

			events := rec.drain()
			require.NotEmpty(t, events)
			require.Equal(t, *tc.expect, events[0].Type())
		})
	}
}

func TestNotifyWorkflowCompleted_EmitsNotPassedFollowUp(t *testing.T) {
	rec := &eventRecorder{}
	status := testkube.FAILED_TestWorkflowStatus
	exec := &testkube.TestWorkflowExecution{
		Id: "exec",
		Result: &testkube.TestWorkflowResult{
			Status:          &status,
			PredictedStatus: &status,
			FinishedAt:      time.Now(),
		},
	}

	notifyWorkflowCompleted(rec, exec)

	events := rec.drain()
	require.GreaterOrEqual(t, len(events), 2)
	require.Equal(t, *testkube.EventEndTestWorkflowFailed, events[0].Type())
	require.Equal(t, *testkube.EventEndTestWorkflowNotPassed, events[1].Type())
}
