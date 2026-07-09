package controlplaneclient

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
)

// logNotification builds a resumable notification whose approximate byte size is
// dominated by the log payload length.
func logNotification(payload string) *testkube.TestWorkflowExecutionNotification {
	return &testkube.TestWorkflowExecutionNotification{
		Ts:  time.Now(),
		Log: payload,
	}
}

// newSessionWithBudget attaches a fresh session to a budget and registers a
// manager-style evictor over the given session set so the budget can reclaim.
func fillSession(s *notificationStreamSession, events int, payload string) {
	for i := 0; i < events; i++ {
		s.publish(logNotification(payload))
	}
}

func budgetUsed(b *liveLogReplayBudget) int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.used
}

func sessionReplayLen(s *notificationStreamSession) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.replay)
}

// TestBudgetEvictsDoneSessionsFirst verifies that when the budget is exceeded,
// the oldest done sessions are dropped to free bytes before running sessions are
// touched.
func TestBudgetEvictsDoneSessionsFirst(t *testing.T) {
	payload := makePayload(4 * 1024)
	// Budget fits roughly two sessions worth of a handful of events.
	budget := newLiveLogReplayBudget(60 * 1024)

	m := newNotificationStreamSessionManager[*fakeNotificationRequest](
		budget,
		func(r *fakeNotificationRequest) string { return r.key },
		nil,
	)

	doneSession := newNotificationStreamSession(budget)
	m.sessions["done"] = doneSession
	fillSession(doneSession, 8, payload)
	doneSession.close(false)

	usedAfterDone := budgetUsed(budget)
	require.Greater(t, usedAfterDone, int64(0))

	runningSession := newNotificationStreamSession(budget)
	m.sessions["running"] = runningSession

	// Push enough into the running session to exceed the budget and force eviction.
	fillSession(runningSession, 12, payload)

	assert.LessOrEqual(t, budgetUsed(budget), budget.max, "used must stay within max after eviction")
	assert.Equal(t, 0, sessionReplayLen(doneSession), "done session buffer should be dropped first")
	_, ok := m.sessions["done"]
	assert.False(t, ok, "done session should be removed from the manager")
	assert.Greater(t, sessionReplayLen(runningSession), 0, "running session should keep streaming")
}

// TestBudgetTrimsRunningSessionsWhenOverBudget verifies that with only running
// sessions present, the oldest events are trimmed so used stays within max.
func TestBudgetTrimsRunningSessionsWhenOverBudget(t *testing.T) {
	payload := makePayload(4 * 1024)
	budget := newLiveLogReplayBudget(40 * 1024)

	m := newNotificationStreamSessionManager[*fakeNotificationRequest](
		budget,
		func(r *fakeNotificationRequest) string { return r.key },
		nil,
	)

	a := newNotificationStreamSession(budget)
	b := newNotificationStreamSession(budget)
	m.sessions["a"] = a
	m.sessions["b"] = b

	fillSession(a, 20, payload)
	fillSession(b, 20, payload)

	assert.LessOrEqual(t, budgetUsed(budget), budget.max, "used must stay within max via trimming")
	// Neither session was done, so both remain registered and still hold some replay.
	_, okA := m.sessions["a"]
	_, okB := m.sessions["b"]
	assert.True(t, okA)
	assert.True(t, okB)
	assert.Greater(t, sessionReplayLen(a)+sessionReplayLen(b), 0, "streaming continues with reduced retention")
}

// TestBudgetAccountingReturnsToZero verifies that releasing every session returns
// the budget used counter to exactly zero.
func TestBudgetAccountingReturnsToZero(t *testing.T) {
	payload := makePayload(2 * 1024)
	budget := newLiveLogReplayBudget(1 * 1024 * 1024)

	m := newNotificationStreamSessionManager[*fakeNotificationRequest](
		budget,
		func(r *fakeNotificationRequest) string { return r.key },
		nil,
	)

	keys := []string{"s1", "s2", "s3"}
	for _, k := range keys {
		s := newNotificationStreamSession(budget)
		m.sessions[k] = s
		fillSession(s, 5, payload)
	}
	require.Greater(t, budgetUsed(budget), int64(0))

	m.mu.Lock()
	for k, s := range m.sessions {
		m.dropSessionLocked(k, s)
	}
	m.mu.Unlock()

	assert.Equal(t, int64(0), budgetUsed(budget), "used must return to zero after all sessions drop")
	assert.Len(t, m.sessions, 0)
}

func makePayload(n int) string {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = 'x'
	}
	return string(buf)
}

type fakeNotificationRequest struct {
	key              string
	streamID         string
	resumeAfterSeqNo uint32
}

func (r *fakeNotificationRequest) GetRequestType() cloud.TestWorkflowNotificationsRequestType {
	return 0
}
func (r *fakeNotificationRequest) GetStreamId() string         { return r.streamID }
func (r *fakeNotificationRequest) GetResumeAfterSeqNo() uint32 { return r.resumeAfterSeqNo }
