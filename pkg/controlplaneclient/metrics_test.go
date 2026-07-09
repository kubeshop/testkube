package controlplaneclient

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

func TestLiveLogMetricsMoveOnAttachAndPublish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	const kind = "metrics-unit-test"

	sourceReady := make(chan channels.WritableWatcher[*testkube.TestWorkflowExecutionNotification], 1)
	manager := newNotificationStreamSessionManager(
		kind,
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			sourceReady <- watcher
			return watcher
		},
	)

	createdBefore := testutil.ToFloat64(liveLogSessionsCreatedTotal.WithLabelValues(kind))

	session, sub, _, _, _, _ := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})

	require.Equal(t, createdBefore+1, testutil.ToFloat64(liveLogSessionsCreatedTotal.WithLabelValues(kind)))
	require.Equal(t, float64(1), testutil.ToFloat64(liveLogSessions.WithLabelValues(kind, "active")))
	require.Equal(t, float64(1), testutil.ToFloat64(liveLogSubscribers.WithLabelValues(kind)))

	// A resumable notification adds to the replay buffer and the memory gauge.
	session.publish(&testkube.TestWorkflowExecutionNotification{Log: "hello"})
	require.Greater(t, testutil.ToFloat64(liveLogReplayBytes.WithLabelValues(kind)), float64(0))

	// Resuming the live session within replay range is available and counted.
	availableBefore := testutil.ToFloat64(liveLogResumeTotal.WithLabelValues(kind, "available"))
	_, sub2, _, available, _, _ := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 1})
	require.True(t, available)
	require.Equal(t, availableBefore+1, testutil.ToFloat64(liveLogResumeTotal.WithLabelValues(kind, "available")))

	watcher := <-sourceReady
	watcher.Close(nil)

	// The source ends: active drops, done rises, and a duration sample is recorded.
	require.Eventually(t, func() bool {
		return testutil.ToFloat64(liveLogSessions.WithLabelValues(kind, "active")) == 0 &&
			testutil.ToFloat64(liveLogSessions.WithLabelValues(kind, "done")) == 1
	}, 2*time.Second, 5*time.Millisecond)
	require.GreaterOrEqual(t, testutil.CollectAndCount(liveLogSourceDurationSeconds), 1)

	manager.detach(session, sub)
	manager.detach(session, sub2)
	require.Equal(t, float64(0), testutil.ToFloat64(liveLogSubscribers.WithLabelValues(kind)))
}

// Two managers of the same kind can overlap: on gRPC reconnect a fresh manager
// is built while the previous one's sessions keep draining bytes via stale TTL
// timers. The replay_bytes gauge must reflect the current live total across
// both, not be clobbered to a single manager's draining value.
func TestLiveLogReplayBytesAdditiveAcrossManagers(t *testing.T) {
	const kind = "metrics-overlap-test"

	newManager := func() *notificationStreamSessionManager[*cloud.TestWorkflowNotificationsRequest] {
		return newNotificationStreamSessionManager(
			kind,
			func(req *cloud.TestWorkflowNotificationsRequest) string { return req.ExecutionId },
			func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
				return channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			},
		)
	}

	require.Equal(t, float64(0), testutil.ToFloat64(liveLogReplayBytes.WithLabelValues(kind)))

	// Old manager buffers some bytes.
	oldManager := newManager()
	oldSession := newNotificationStreamSession(oldManager.addReplayBytes)
	oldSession.publish(&testkube.TestWorkflowExecutionNotification{Log: "old-manager-payload"})
	oldBytes := testutil.ToFloat64(liveLogReplayBytes.WithLabelValues(kind))
	require.Greater(t, oldBytes, float64(0))

	// A reconnect builds a fresh manager whose session buffers more bytes.
	newMgr := newManager()
	newSession := newNotificationStreamSession(newMgr.addReplayBytes)
	newSession.publish(&testkube.TestWorkflowExecutionNotification{Log: "new-manager-payload"})
	combined := testutil.ToFloat64(liveLogReplayBytes.WithLabelValues(kind))
	require.Greater(t, combined, oldBytes)

	newBytes := combined - oldBytes

	// The old manager's stale timer releases its session's bytes. The gauge must
	// drop by exactly the old contribution and still hold the new manager's live
	// total, not be reset to the old manager's (now zero) draining value.
	oldSession.releaseReplayBytes()
	require.Equal(t, newBytes, testutil.ToFloat64(liveLogReplayBytes.WithLabelValues(kind)))

	// Once the new manager's session drops too, the gauge nets back to zero.
	newSession.releaseReplayBytes()
	require.Equal(t, float64(0), testutil.ToFloat64(liveLogReplayBytes.WithLabelValues(kind)))
}
