package controlplaneclient

import (
	"sync"
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

// sumReplayBytes returns the actual bytes currently held across the given
// sessions, read under each session lock.
func sumReplayBytes(sessions ...*notificationStreamSession) int64 {
	var total int64
	for _, s := range sessions {
		s.mu.Lock()
		total += int64(s.replayBytes)
		s.mu.Unlock()
	}
	return total
}

// TestBudgetNoPhantomOvercountUnderConcurrency stresses the budget with many
// publishers over sessions that share an over-tight budget, forcing continuous
// eviction. It asserts used never drifts above the actual bytes held: with the
// reservation happening under the session lock, a concurrent evictor can only
// release bytes already reserved, so used stays consistent. Against the old
// reserve-after-unlock ordering, an evictor could release a not-yet-reserved
// growth, leaving a phantom overcount that this test detects.
func TestBudgetNoPhantomOvercountUnderConcurrency(t *testing.T) {
	payload := makePayload(2 * 1024)
	// Tight enough that every publisher forces eviction across the shared budget.
	budget := newLiveLogReplayBudget(24 * 1024)

	m := newNotificationStreamSessionManager[*fakeNotificationRequest](
		budget,
		func(r *fakeNotificationRequest) string { return r.key },
		nil,
	)

	const sessionCount = 4
	sessions := make([]*notificationStreamSession, sessionCount)
	for i := 0; i < sessionCount; i++ {
		s := newNotificationStreamSession(budget)
		sessions[i] = s
		m.mu.Lock()
		m.sessions[string(rune('a'+i))] = s
		m.mu.Unlock()
	}

	const publishers = 8
	const perPublisher = 400
	var wg sync.WaitGroup
	for p := 0; p < publishers; p++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			for i := 0; i < perPublisher; i++ {
				sessions[(p+i)%sessionCount].publish(logNotification(payload))
			}
		}(p)
	}
	wg.Wait()

	used := budgetUsed(budget)
	actual := sumReplayBytes(sessions...)
	assert.Equal(t, actual, used, "used must equal the actual bytes held across live sessions (no phantom overcount)")
	assert.LessOrEqual(t, used, budget.max, "used must not exceed max after eviction settles")
}

func budgetEvictorCount(b *liveLogReplayBudget) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.evictors)
}

// TestBudgetEvictorsDeregisterOnManagerStop verifies that managers registering
// against a shared, long-lived budget do not leak evictors: after each manager
// stops, the budget's evictor set returns to its baseline instead of growing with
// every reconnect.
func TestBudgetEvictorsDeregisterOnManagerStop(t *testing.T) {
	budget := newLiveLogReplayBudget(1 * 1024 * 1024)
	require.Equal(t, 0, budgetEvictorCount(budget), "fresh budget has no evictors")

	const managers = 50
	stops := make([]func(), 0, managers)
	for i := 0; i < managers; i++ {
		m := newNotificationStreamSessionManager[*fakeNotificationRequest](
			budget,
			func(r *fakeNotificationRequest) string { return r.key },
			nil,
		)
		stops = append(stops, m.stop)
	}
	require.Equal(t, managers, budgetEvictorCount(budget), "each manager registers one evictor")

	for _, stop := range stops {
		stop()
	}
	assert.Equal(t, 0, budgetEvictorCount(budget), "stopping every manager returns the evictor set to baseline")
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
