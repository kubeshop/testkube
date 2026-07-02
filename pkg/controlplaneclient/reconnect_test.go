package controlplaneclient

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

// readNotificationSeqNos reads exactly n live events off a subscription channel,
// returning the seqNo it observed for each. It fails the test on timeout so a
// lost/dropped event surfaces as a failure rather than a hang.
func readNotificationSeqNos(t *testing.T, sub *notificationStreamSubscription, n int) []uint32 {
	t.Helper()
	seqNos := make([]uint32, 0, n)
	for i := 0; i < n; i++ {
		select {
		case event := <-sub.ch:
			seqNos = append(seqNos, event.seqNo)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for live notification %d/%d (got %v)", i+1, n, seqNos)
		}
	}
	return seqNos
}

// TestNotificationStreamSessionManager_ReconnectResumesWithoutGapOrDuplicate drives the
// session manager directly, dropping a subscription in place of a real gRPC stream break, and
// checks that the resume protocol in notificationStreamSessionManager.attach replays exactly
// the missed sequence range against the same live session, with no duplication and no loss. A
// fresh session runs runSource, which pumps a channels.Watcher into
// notificationStreamSession.publish, assigning seqNos and filling the replay ring. The
// reconnect reuses that live session and reads the gap from replayAfterLocked.
func TestNotificationStreamSessionManager_ReconnectResumesWithoutGapOrDuplicate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	const streamID = "stream-1"

	// Source coordination. The live source for streamID emits seq 1..5 immediately, then
	// blocks until the test asks for one more live event (post-reconnect), then blocks again
	// until release so the session stays LIVE (not done) across the disconnect/reconnect.
	emitSixth := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	manager := newNotificationStreamSessionManager(
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(_ context.Context, req *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			// Only the primary stream produces data. Any other stream id (the resume-unavailable
			// probe below) just parks until release so it never injects spurious sequence numbers.
			if req.StreamId != streamID {
				go func() {
					<-release
					watcher.Close(nil)
				}()
				return watcher
			}
			go func() {
				for i := 1; i <= 5; i++ {
					watcher.Send(&testkube.TestWorkflowExecutionNotification{
						Ts:  time.Now(),
						Log: fmt.Sprintf("log-%d", i),
					})
				}
				<-emitSixth
				watcher.Send(&testkube.TestWorkflowExecutionNotification{
					Ts:  time.Now(),
					Log: "log-6",
				})
				<-release
				watcher.Close(nil)
			}()
			return watcher
		},
	)

	// 1. Fresh attach with ResumeAfterSeqNo=0 starts a brand new session. The source publishes
	//    seq 1..5 and the subscriber must observe all of them in order.
	session1, sub1, replay1, available1, _, done1 := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{
		ExecutionId: "exec-1",
		StreamId:    streamID,
	})
	require.True(t, available1, "fresh non-resuming attach is trivially resume-available")
	require.False(t, done1, "session must be live")
	require.Empty(t, replay1, "fresh attach has nothing to replay")

	firstPass := readNotificationSeqNos(t, sub1, 5)
	assert.Equal(t, []uint32{1, 2, 3, 4, 5}, firstPass, "subscriber must receive the live sequence 1..5")

	// All five must be durably recorded in the replay ring before we reconnect.
	require.Eventually(t, func() bool {
		return session1.currentSeqNo() == 5
	}, 2*time.Second, time.Millisecond, "replay ring must hold up to seq 5")

	// 2. DISCONNECT: the client has fallen behind at seq 3 (frames 4 and 5 were in flight and
	//    are treated as not yet acked). Unsubscribing the session subscription directly stands
	//    in for a dropped subscription; it exercises the session manager's attach/replay/resume
	//    bookkeeping, not the real gRPC processNotifications stream-break loop.
	session1.unsubscribe(sub1)
	waitForNotificationSubscriptionDone(t, sub1)

	// 3. RECONNECT: same streamId, ResumeAfterSeqNo=3. Must REUSE the live session and replay
	//    exactly the gap (seq 4 and 5). No duplication of 1..3, no loss.
	session2, sub2, replay2, available2, lastSeqNo2, done2 := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{
		ExecutionId:      "exec-1",
		StreamId:         streamID,
		ResumeAfterSeqNo: 3,
	})
	t.Cleanup(func() { session2.unsubscribe(sub2) })

	require.Same(t, session1, session2, "reconnect must reuse the live session, not start a duplicate")
	require.True(t, available2, "the resume point is in the replay ring, so resume is available")
	require.False(t, done2, "session is still live")
	require.Equal(t, uint32(5), lastSeqNo2, "manager reports the live tail seqNo")
	assert.Equal(t, []uint32{4, 5}, collectReplaySeqNos(replay2), "replay must be exactly the missed gap 4,5")

	// Continuity: a brand new subscriber on the reused session must not also receive 1..3
	// from its live channel (those only came back as replay). The next live event is seq 6.
	emitSixth <- struct{}{}
	live := readNotificationSeqNos(t, sub2, 1)
	assert.Equal(t, []uint32{6}, live, "post-reconnect live delivery continues at seq 6 with no duplicate replay on the channel")

	// 4. RESUME_UNAVAILABLE: attaching with ResumeAfterSeqNo>0 to a streamId that has no live
	//    session yields a fresh session whose resume point cannot be satisfied. attach reports
	//    available=false, which drives the WORKFLOW_STREAM_RESUME_UNAVAILABLE protocol frame in
	//    processNotifications.
	session3, sub3, replay3, available3, lastSeqNo3, done3 := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{
		ExecutionId:      "exec-1",
		StreamId:         "stream-resume-unavailable",
		ResumeAfterSeqNo: 4,
	})
	t.Cleanup(func() { session3.unsubscribe(sub3) })

	require.NotSame(t, session1, session3, "an unknown stream must not reuse another stream's session")
	require.False(t, available3, "resume is NOT available -> RESUME_UNAVAILABLE must be signalled")
	require.False(t, done3, "the fresh session is live, just not resumable")
	require.Zero(t, lastSeqNo3, "fresh session has produced nothing yet")
	require.Empty(t, replay3, "nothing to replay for an unsatisfiable resume point")
}
