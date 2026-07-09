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
