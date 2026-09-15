package controlplaneclient

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/internal/config"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

func TestListTestWorkflows_ForwardsOptionsLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockCloudClient := cloud.NewMockTestKubeCloudAPIClient(ctrl)
	client := &client{
		client:     mockCloudClient,
		proContext: config.ProContext{},
	}

	expectedErr := errors.New("list failed")
	mockCloudClient.EXPECT().
		ListTestWorkflows(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *cloud.ListTestWorkflowsRequest, _ ...grpc.CallOption) (cloud.TestKubeCloudAPI_ListTestWorkflowsClient, error) {
			require.Equal(t, uint32(25), req.Offset)
			require.Equal(t, uint32(250), req.Limit)
			require.Equal(t, map[string]string{"team": "qa"}, req.Labels)
			require.Equal(t, "smoke", req.TextSearch)
			return nil, expectedErr
		})

	items, err := client.ListTestWorkflows(context.Background(), "env-1", ListTestWorkflowOptions{
		Offset:     25,
		Limit:      250,
		Labels:     map[string]string{"team": "qa"},
		TextSearch: "smoke",
	}).All()

	require.ErrorIs(t, err, expectedErr)
	require.Empty(t, items)
}

func TestListTestWorkflows_LeavesLimitUnsetWhenOptionIsZero(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockCloudClient := cloud.NewMockTestKubeCloudAPIClient(ctrl)
	client := &client{
		client:     mockCloudClient,
		proContext: config.ProContext{},
	}

	expectedErr := errors.New("list failed")
	mockCloudClient.EXPECT().
		ListTestWorkflows(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *cloud.ListTestWorkflowsRequest, _ ...grpc.CallOption) (cloud.TestKubeCloudAPI_ListTestWorkflowsClient, error) {
			require.Equal(t, uint32(0), req.Limit)
			return nil, expectedErr
		})

	_, err := client.ListTestWorkflows(context.Background(), "env-1", ListTestWorkflowOptions{}).All()

	require.ErrorIs(t, err, expectedErr)
}

func TestNotificationStreamSessionManagerReplaysAfterCursor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	manager := newNotificationStreamSessionManager(
		ctx,
		"workflow",
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			go func() {
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "one"})
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "two"})
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "three"})
				watcher.Close(nil)
			}()
			return watcher
		},
	)

	session, sub, replay, available, _, done := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	require.True(t, available)
	require.False(t, done)
	require.Empty(t, replay)

	firstPass := collectNotificationSubscriptionSeqNos(t, sub)
	session.unsubscribe(sub)

	assert.Equal(t, []uint32{1, 2, 3}, firstPass)

	session, sub, replay, available, lastSeqNo, done := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 1})
	require.True(t, available)
	require.True(t, done)
	require.Equal(t, uint32(3), lastSeqNo)
	require.Len(t, replay, 2)
	assert.Equal(t, []uint32{2, 3}, []uint32{replay[0].seqNo, replay[1].seqNo})
	session.unsubscribe(sub)
}

func TestNotificationStreamSessionSurvivesReaderDropAndResumesLive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	emit := make(chan string)
	manager := newNotificationStreamSessionManager(
		ctx,
		"workflow",
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			go func() {
				for log := range emit {
					watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: log})
				}
				watcher.Close(nil)
			}()
			return watcher
		},
	)

	// First reader attaches; the source starts producing.
	session, sub, _, available, _, done := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	require.True(t, available)
	require.False(t, done)
	emit <- "a"
	emit <- "b"
	require.Eventually(t, func() bool { return session.currentSeqNo() == 2 }, time.Second, 5*time.Millisecond)

	// The reader disconnects, but the source keeps running under the manager context.
	session.unsubscribe(sub)
	emit <- "c"
	emit <- "d"
	require.Eventually(t, func() bool { return session.currentSeqNo() == 4 }, time.Second, 5*time.Millisecond)

	// A reconnect with the same streamId resumes from the cursor, not resume_unavailable.
	session2, sub2, replay, available2, lastSeqNo, done2 := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 2})
	require.True(t, available2, "resume must be available: the source survived the reader disconnect")
	require.False(t, done2)
	require.Equal(t, uint32(4), lastSeqNo)
	require.Len(t, replay, 2)
	assert.Equal(t, []uint32{3, 4}, []uint32{replay[0].seqNo, replay[1].seqNo})

	session2.unsubscribe(sub2)
	close(emit)
}

func TestSendNotificationResponseReturnsContextErrorWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	responses := make(chan string)
	err := sendNotificationResponse(ctx, responses, "response")

	require.ErrorIs(t, err, context.Canceled)
}

func TestNotificationStreamSessionPublishDoesNotHoldLockForSlowSubscriber(t *testing.T) {
	session := newNotificationStreamSession(defaultNotificationReplayLimits())
	sub, _, _, _, _ := session.subscribe(0, 1)
	for i := 0; i < cap(sub.ch); i++ {
		sub.ch <- notificationStreamEvent{}
	}

	publishDone := make(chan struct{})
	go func() {
		session.publish(&testkube.TestWorkflowExecutionNotification{Log: "blocked"})
		close(publishDone)
	}()

	require.Eventually(t, func() bool {
		return session.currentSeqNo() == 1
	}, time.Second, time.Millisecond)

	unsubscribeDone := make(chan struct{})
	go func() {
		session.unsubscribe(sub)
		close(unsubscribeDone)
	}()

	select {
	case <-unsubscribeDone:
	case <-time.After(time.Second):
		t.Fatal("unsubscribe blocked while publish was waiting on a slow subscriber")
	}

	select {
	case <-publishDone:
	case <-time.After(time.Second):
		t.Fatal("publish did not finish after subscriber was closed")
	}
}

func TestWorkflowProtocolEventsDoNotAdvanceApplicationSeqNo(t *testing.T) {
	session := newNotificationStreamSession(defaultNotificationReplayLimits())

	ready := buildCloudProtocol("stream-1", session.currentSeqNo(), cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_READY, "")
	require.Equal(t, uint32(0), ready.SeqNo)
	require.Equal(t, uint32(0), session.currentSeqNo())

	session.publish(&testkube.TestWorkflowExecutionNotification{Log: "application log"})
	require.Equal(t, uint32(1), session.currentSeqNo())

	heartbeat := buildCloudProtocol("stream-1", session.currentSeqNo(), cloud.TestWorkflowNotificationType_WORKFLOW_STREAM_HEARTBEAT, "")
	require.Equal(t, uint32(1), heartbeat.SeqNo)
	require.Equal(t, uint32(1), session.currentSeqNo())
}

func TestNotificationStreamSessionReplayUnavailableForTrimmedCursor(t *testing.T) {
	session := newNotificationStreamSession(defaultNotificationReplayLimits())
	for i := 0; i < workflowNotificationReplayMaxEvents+2; i++ {
		session.publish(&testkube.TestWorkflowExecutionNotification{Log: "log"})
	}

	sub, replay, available, lastSeqNo, done := session.subscribe(1, 1)
	t.Cleanup(func() {
		session.unsubscribe(sub)
	})

	require.False(t, available)
	require.False(t, done)
	require.Empty(t, replay)
	require.Equal(t, uint32(workflowNotificationReplayMaxEvents+2), lastSeqNo)
}

func TestNotificationStreamSessionManagerStartsFreshAfterDoneSessionWithoutResume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var processCalls atomic.Int32
	manager := newNotificationStreamSessionManager(
		ctx,
		"workflow",
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			processCalls.Add(1)
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			go func() {
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "done"})
				watcher.Close(nil)
			}()
			return watcher
		},
	)

	session1, sub1, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	waitForNotificationSubscriptionDone(t, sub1)
	session1.unsubscribe(sub1)

	session2, sub2, replay, available, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	waitForNotificationSubscriptionDone(t, sub2)
	session2.unsubscribe(sub2)

	require.NotSame(t, session1, session2)
	require.True(t, available)
	require.Empty(t, replay)
	require.Equal(t, int32(2), processCalls.Load())
}

func TestNotificationStreamSessionManagerStartsFreshAfterErroredDoneSessionWithResume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	releaseSecond := make(chan struct{})
	var processCalls atomic.Int32
	manager := newNotificationStreamSessionManager(
		ctx,
		"workflow",
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			call := processCalls.Add(1)
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			go func() {
				if call == 1 {
					watcher.Send(&testkube.TestWorkflowExecutionNotification{Ts: time.Now(), Log: "attempt"})
					watcher.Close(errors.New("source failed"))
					return
				}
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Ts: time.Now().Add(-time.Minute), Log: "historical"})
				<-releaseSecond
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Ts: time.Now(), Log: "live"})
				watcher.Close(nil)
			}()
			return watcher
		},
	)

	session1, sub1, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	firstPass := collectNotificationSubscriptionSeqNos(t, sub1)
	session1.unsubscribe(sub1)
	require.NotEmpty(t, firstPass)

	session2, sub2, replay, available, lastSeqNo, done := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: firstPass[len(firstPass)-1]})
	defer session2.unsubscribe(sub2)

	require.NotSame(t, session1, session2)
	require.False(t, available)
	require.False(t, done)
	require.Zero(t, lastSeqNo)
	close(releaseSecond)

	secondPass := collectReplaySeqNos(replay)
	if len(secondPass) == 0 {
		select {
		case event := <-sub2.ch:
			secondPass = append(secondPass, event.seqNo)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for resumed notification")
		}
	}

	require.Equal(t, []uint32{1}, secondPass)
	require.Eventually(t, func() bool {
		return processCalls.Load() == 2
	}, 2*time.Second, time.Millisecond)
}

func TestNotificationStreamSessionManagerFreshResumeStartsFromLiveTail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	release := make(chan struct{})
	manager := newNotificationStreamSessionManager(
		ctx,
		"workflow",
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			go func() {
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Ts: time.Now().Add(-time.Minute), Log: "historical"})
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "temporary", Temporary: true})
				<-release
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Ts: time.Now(), Log: "live"})
				watcher.Close(nil)
			}()
			return watcher
		},
	)

	session, sub, replay, available, lastSeqNo, done := manager.attach(&cloud.TestWorkflowNotificationsRequest{
		ExecutionId:      "exec-1",
		StreamId:         "stream-1",
		ResumeAfterSeqNo: 7,
	})
	defer session.unsubscribe(sub)

	require.False(t, available)
	require.False(t, done)
	require.Zero(t, lastSeqNo)
	require.Empty(t, replay)
	close(release)

	select {
	case event := <-sub.ch:
		require.Equal(t, uint32(1), event.seqNo)
		require.Equal(t, "live", event.notification.Log)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for live-tail notification")
	}
}

func TestNotificationStreamSessionManagerMarksResumeUnavailableForFreshSessionWithResume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	release := make(chan struct{})
	manager := newNotificationStreamSessionManager(
		ctx,
		"workflow",
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			go func() {
				<-release
				watcher.Close(nil)
			}()
			return watcher
		},
	)

	session, sub, replay, available, lastSeqNo, done := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 7})
	defer session.unsubscribe(sub)
	defer close(release)

	require.False(t, available)
	require.False(t, done)
	require.Zero(t, lastSeqNo)
	require.Empty(t, replay)
}

func TestNotificationStreamSessionManagerStartsFreshForConcurrentViewersWithoutResume(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	releaseFirst := make(chan struct{})
	releaseSecond := make(chan struct{})
	var processCalls atomic.Int32
	manager := newNotificationStreamSessionManager(
		ctx,
		"workflow",
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(_ context.Context, req *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			call := processCalls.Add(1)
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			go func() {
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: req.StreamId})
				if call == 1 {
					<-releaseFirst
				} else {
					<-releaseSecond
				}
				watcher.Close(nil)
			}()
			return watcher
		},
	)

	session1, sub1, _, available1, _, done1 := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	defer session1.unsubscribe(sub1)
	session2, sub2, replay2, available2, _, done2 := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-2"})
	defer session2.unsubscribe(sub2)
	defer close(releaseFirst)
	defer close(releaseSecond)

	require.NotSame(t, session1, session2)
	require.True(t, available1)
	require.False(t, done1)
	require.True(t, available2)
	require.False(t, done2)
	require.Empty(t, replay2)
	require.Eventually(t, func() bool {
		return processCalls.Load() == 2
	}, 2*time.Second, time.Millisecond)

	select {
	case event := <-sub2.ch:
		require.Equal(t, "stream-2", event.notification.Log)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second viewer initial notification")
	}
}

func TestNotificationStreamSessionManagerExpiresDoneSessionsWithoutAttach(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	release := make(chan struct{})
	manager := newNotificationStreamSessionManager(
		ctx,
		"workflow",
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			go func() {
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "done"})
				<-release
				watcher.Close(nil)
			}()
			return watcher
		},
	)
	manager.sessionIdleTTL = 10 * time.Millisecond

	session, sub, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	close(release)
	waitForNotificationSubscriptionDone(t, sub)
	session.unsubscribe(sub)

	require.Eventually(t, func() bool {
		manager.mu.Lock()
		defer manager.mu.Unlock()
		return len(manager.sessions) == 0
	}, time.Second, time.Millisecond)
}

func TestNotificationStreamSessionManagerSweepExpiredRemovesDonePastTTLSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	release := make(chan struct{})
	manager := newNotificationStreamSessionManager(
		ctx,
		"workflow",
		func(req *cloud.TestWorkflowNotificationsRequest) string {
			return req.ExecutionId
		},
		func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			go func() {
				watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "done"})
				<-release
				watcher.Close(nil)
			}()
			return watcher
		},
	)
	manager.sessionIdleTTL = time.Minute

	session, sub, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	close(release)
	waitForNotificationSubscriptionDone(t, sub)
	session.unsubscribe(sub)

	require.Eventually(t, func() bool {
		done, _ := session.status()
		return done
	}, time.Second, time.Millisecond)

	manager.mu.Lock()
	require.Len(t, manager.sessions, 1)
	manager.mu.Unlock()

	// A sweep at now-before-TTL leaves the session in place.
	manager.sweepExpired(time.Now())
	manager.mu.Lock()
	require.Len(t, manager.sessions, 1)
	manager.mu.Unlock()

	// A sweep at a time past the TTL reclaims it with no further attach traffic.
	manager.sweepExpired(time.Now().Add(2 * manager.sessionIdleTTL))
	manager.mu.Lock()
	require.Len(t, manager.sessions, 0)
	manager.mu.Unlock()
}

func collectNotificationSubscriptionSeqNos(t *testing.T, sub *notificationStreamSubscription) []uint32 {
	t.Helper()

	deadline := time.After(2 * time.Second)
	var seqNos []uint32
	for {
		select {
		case event := <-sub.ch:
			seqNos = append(seqNos, event.seqNo)
		case <-sub.done:
			for {
				select {
				case event := <-sub.ch:
					seqNos = append(seqNos, event.seqNo)
				default:
					return seqNos
				}
			}
		case <-deadline:
			t.Fatal("timed out waiting for notification subscription to finish")
			return nil
		}
	}
}

func collectReplaySeqNos(replay []notificationStreamEvent) []uint32 {
	seqNos := make([]uint32, 0, len(replay))
	for _, event := range replay {
		seqNos = append(seqNos, event.seqNo)
	}
	return seqNos
}

func waitForNotificationSubscriptionDone(t *testing.T, sub *notificationStreamSubscription) {
	t.Helper()

	select {
	case <-sub.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification subscription to finish")
	}
}

func TestNotificationStreamSessionReplacementCancelsOrphanedSource(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	sourceCtxs := make(chan context.Context, 4)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	manager := newNotificationStreamSessionManager(
		ctx,
		"workflow",
		func(req *cloud.TestWorkflowNotificationsRequest) string { return req.ExecutionId },
		func(sourceCtx context.Context, _ *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
			sourceCtxs <- sourceCtx
			watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
			go func() {
				select {
				case <-sourceCtx.Done():
				case <-release:
				}
				watcher.Close(nil)
			}()
			return watcher
		},
	)

	req := &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"}
	session1, sub1, _, _, _, _ := manager.attach(req)
	t.Cleanup(func() { session1.unsubscribe(sub1) })

	var firstSource context.Context
	select {
	case firstSource = <-sourceCtxs:
	case <-time.After(2 * time.Second):
		t.Fatal("first source was not started")
	}

	// A resume-from-zero attach with the same key replaces the session; the old
	// source must be cancelled instead of running orphaned until execution end.
	session2, sub2, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{
		ExecutionId:      "exec-1",
		StreamId:         "stream-1",
		ResumeAfterSeqNo: 0,
	})
	t.Cleanup(func() { session2.unsubscribe(sub2) })
	require.NotSame(t, session1, session2, "resume-from-zero must create a new session")

	select {
	case <-firstSource.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("replaced session source context was not cancelled")
	}
}

type workflowSessionManager = notificationStreamSessionManager[*cloud.TestWorkflowNotificationsRequest]

// newMetricsTestSessionManager builds a manager whose kind label is the test name,
// so the counter series it raises on the shared default registry are easy to tell
// apart. The gauges are read through liveLogStats and touch no shared state.
func newMetricsTestSessionManager(t *testing.T, ctx context.Context, process func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher) *workflowSessionManager {
	t.Helper()

	manager := newNotificationStreamSessionManager(
		ctx,
		t.Name(),
		func(req *cloud.TestWorkflowNotificationsRequest) string { return req.ExecutionId },
		process,
	)
	manager.sessionIdleTTL = time.Minute
	return manager
}

// silentNotificationSource returns a process function whose watcher sends nothing
// and ends when the test finishes the execution id with an error, or when the
// manager cancels the source, the way a replaced session's source ends.
func silentNotificationSource(t *testing.T) (func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher, func(executionID string, err error)) {
	t.Helper()

	type sourceEnd struct {
		err  error
		done chan struct{}
	}
	var mu sync.Mutex
	ends := make(map[string]*sourceEnd)
	process := func(sourceCtx context.Context, req *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
		end := &sourceEnd{done: make(chan struct{})}
		mu.Lock()
		ends[req.ExecutionId] = end
		mu.Unlock()
		watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
		go func() {
			select {
			case <-end.done:
				watcher.Close(end.err)
			case <-sourceCtx.Done():
				watcher.Close(nil)
			}
		}()
		return watcher
	}
	// finish waits for the source, because runSource starts it on its own goroutine.
	finish := func(executionID string, err error) {
		var end *sourceEnd
		require.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			end = ends[executionID]
			return end != nil
		}, 2*time.Second, time.Millisecond, "no source was started for %q", executionID)
		end.err = err
		close(end.done)
	}
	return process, finish
}

// liveLogCounters is the growth of every live-log counter series of one kind.
type liveLogCounters struct {
	created           float64
	resumeAvailable   float64
	resumeUnavailable float64
	evictedTTL        float64
	evictedErrored    float64
	evictedReplaced   float64
	sourcesEndedOK    float64
	sourcesEndedError float64
}

// liveLogCountersSince returns the counters' growth since the call. The counters
// live on the shared default registry, so a test asserts what it caused and not
// an absolute value that another run with the same label may have raised.
func liveLogCountersSince(t *testing.T, kind string) func() liveLogCounters {
	t.Helper()

	sampleCount := func(result string) float64 {
		var metric dto.Metric
		require.NoError(t, liveLogSourceDurationSeconds.WithLabelValues(kind, result).(prometheus.Metric).Write(&metric))
		return float64(metric.GetHistogram().GetSampleCount())
	}
	read := func() liveLogCounters {
		return liveLogCounters{
			created:           testutil.ToFloat64(liveLogSessionsCreatedTotal.WithLabelValues(kind)),
			resumeAvailable:   testutil.ToFloat64(liveLogResumeTotal.WithLabelValues(kind, "available")),
			resumeUnavailable: testutil.ToFloat64(liveLogResumeTotal.WithLabelValues(kind, "unavailable")),
			evictedTTL:        testutil.ToFloat64(liveLogSessionsEvictedTotal.WithLabelValues(kind, liveLogEvictionReasonTTL)),
			evictedErrored:    testutil.ToFloat64(liveLogSessionsEvictedTotal.WithLabelValues(kind, liveLogEvictionReasonErrored)),
			evictedReplaced:   testutil.ToFloat64(liveLogSessionsEvictedTotal.WithLabelValues(kind, liveLogEvictionReasonReplaced)),
			sourcesEndedOK:    sampleCount(liveLogResultOK),
			sourcesEndedError: sampleCount(liveLogResultError),
		}
	}
	base := read()
	return func() liveLogCounters {
		now := read()
		return liveLogCounters{
			created:           now.created - base.created,
			resumeAvailable:   now.resumeAvailable - base.resumeAvailable,
			resumeUnavailable: now.resumeUnavailable - base.resumeUnavailable,
			evictedTTL:        now.evictedTTL - base.evictedTTL,
			evictedErrored:    now.evictedErrored - base.evictedErrored,
			evictedReplaced:   now.evictedReplaced - base.evictedReplaced,
			sourcesEndedOK:    now.sourcesEndedOK - base.sourcesEndedOK,
			sourcesEndedError: now.sourcesEndedError - base.sourcesEndedError,
		}
	}
}

func notificationBytes(logs ...string) int {
	total := 0
	for _, log := range logs {
		total += approximateNotificationBytes(&testkube.TestWorkflowExecutionNotification{Log: log})
	}
	return total
}

func waitForNotificationSessionDone(t *testing.T, session *notificationStreamSession) {
	t.Helper()

	require.Eventually(t, func() bool {
		done, _ := session.status()
		return done
	}, 2*time.Second, time.Millisecond)
}

// waitForLiveLogMetrics waits for the counters and stats to settle, because a
// source ends on its own goroutine after the test's last call.
func waitForLiveLogMetrics(t *testing.T, manager *workflowSessionManager, counters func() liveLogCounters, wantCounters liveLogCounters, wantStats liveLogStats) {
	t.Helper()

	require.Eventually(t, func() bool {
		return counters() == wantCounters && manager.liveLogStats() == wantStats
	}, 2*time.Second, time.Millisecond, "metrics did not settle: counters %+v stats %+v", counters(), manager.liveLogStats())
}

func TestNotificationStreamSessionManagerMetricsFollowSessionLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	process, finish := silentNotificationSource(t)
	manager := newMetricsTestSessionManager(t, ctx, process)
	counters := liveLogCountersSince(t, manager.kind)

	// A fresh attach creates one active session with one subscriber.
	session, sub, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	want := liveLogCounters{created: 1}
	assert.Equal(t, want, counters())
	assert.Equal(t, liveLogStats{activeSessions: 1, subscribers: 1}, manager.liveLogStats())

	// A second viewer that resumes the same stream shares the session and its buffer.
	session.publish(&testkube.TestWorkflowExecutionNotification{Log: "one"})
	_, sub2, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 1})
	want.resumeAvailable = 1
	assert.Equal(t, want, counters())
	assert.Equal(t, liveLogStats{activeSessions: 1, subscribers: 2, replayBytes: notificationBytes("one")}, manager.liveLogStats())

	// The source ends: the session is done, its subscribers are closed, and its
	// duration is recorded as a success. The replay buffer stays for late viewers.
	finish("exec-1", nil)
	want.sourcesEndedOK = 1
	waitForLiveLogMetrics(t, manager, counters, want, liveLogStats{doneSessions: 1, replayBytes: notificationBytes("one")})
	session.unsubscribe(sub)
	session.unsubscribe(sub2)

	// The idle TTL passes: the sweep evicts the done session and frees its buffer.
	manager.sweepExpired(time.Now().Add(2 * manager.sessionIdleTTL))
	want.evictedTTL = 1
	assert.Equal(t, want, counters())
	assert.Equal(t, liveLogStats{}, manager.liveLogStats())

	// The manager's context ends: it stops reporting to the collector.
	registered := func() bool {
		liveLogMetrics.mu.Lock()
		defer liveLogMetrics.mu.Unlock()
		_, ok := liveLogMetrics.sources[manager]
		return ok
	}
	require.True(t, registered())
	cancel()
	require.Eventually(t, func() bool { return !registered() }, 2*time.Second, time.Millisecond)
}

// priorSessionState is the state of the session a request finds under its key.
type priorSessionState int

const (
	noPriorSession priorSessionState = iota
	runningPriorSession
	donePriorSession
	erroredPriorSession
)

func TestNotificationStreamSessionManagerMetricsFollowAttach(t *testing.T) {
	start := &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"}
	resume := &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 1}
	tests := []struct {
		name         string
		prior        priorSessionState
		req          *cloud.TestWorkflowNotificationsRequest
		wantCounters liveLogCounters
		wantStats    liveLogStats
		// joinsPrior is true when the request must attach to the prior session
		// instead of creating a new one.
		joinsPrior bool
	}{
		{
			name:         "fresh attach creates a session",
			prior:        noPriorSession,
			req:          start,
			wantCounters: liveLogCounters{created: 1},
			wantStats:    liveLogStats{activeSessions: 1, subscribers: 1},
		},
		{
			name:         "resume inside the replay buffer joins the session",
			prior:        runningPriorSession,
			req:          resume,
			wantCounters: liveLogCounters{resumeAvailable: 1},
			wantStats:    liveLogStats{activeSessions: 1, subscribers: 2, replayBytes: notificationBytes("one")},
			joinsPrior:   true,
		},
		{
			name:         "resume without a session starts fresh and is unavailable",
			prior:        noPriorSession,
			req:          resume,
			wantCounters: liveLogCounters{created: 1, resumeUnavailable: 1},
			wantStats:    liveLogStats{activeSessions: 1, subscribers: 1},
		},
		{
			name:         "resume after a failed source evicts the errored session",
			prior:        erroredPriorSession,
			req:          resume,
			wantCounters: liveLogCounters{created: 1, resumeUnavailable: 1, evictedErrored: 1},
			wantStats:    liveLogStats{activeSessions: 1, subscribers: 1},
		},
		{
			name:         "start from zero replaces a done session",
			prior:        donePriorSession,
			req:          start,
			wantCounters: liveLogCounters{created: 1, evictedReplaced: 1},
			wantStats:    liveLogStats{activeSessions: 1, subscribers: 1},
		},
		{
			name:         "start from zero replaces a running session and cancels its source",
			prior:        runningPriorSession,
			req:          start,
			wantCounters: liveLogCounters{created: 1, evictedReplaced: 1, sourcesEndedOK: 1},
			wantStats:    liveLogStats{activeSessions: 1, subscribers: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			process, finish := silentNotificationSource(t)
			manager := newMetricsTestSessionManager(t, ctx, process)

			var prior *notificationStreamSession
			if tt.prior != noPriorSession {
				var sub *notificationStreamSubscription
				prior, sub, _, _, _, _ = manager.attach(start)
				t.Cleanup(func() { prior.unsubscribe(sub) })
				prior.publish(&testkube.TestWorkflowExecutionNotification{Log: "one"})
			}
			switch tt.prior {
			case donePriorSession:
				finish("exec-1", nil)
				waitForNotificationSessionDone(t, prior)
			case erroredPriorSession:
				finish("exec-1", errors.New("source failed"))
				waitForNotificationSessionDone(t, prior)
			}
			counters := liveLogCountersSince(t, manager.kind)

			session, sub, _, _, _, _ := manager.attach(tt.req)
			t.Cleanup(func() { session.unsubscribe(sub) })

			waitForLiveLogMetrics(t, manager, counters, tt.wantCounters, tt.wantStats)
			manager.mu.Lock()
			current := manager.sessions["exec-1:stream-1"]
			manager.mu.Unlock()
			if tt.joinsPrior {
				assert.Same(t, prior, current, "the request must join the prior session")
			} else {
				assert.NotSame(t, prior, current, "the prior session must leave the manager")
			}
		})
	}
}

func TestNotificationStreamSessionReplayBytesFollowTrimming(t *testing.T) {
	session := newNotificationStreamSession(notificationReplayLimits{maxEvents: 3, maxBytes: workflowNotificationReplayMaxBytes})
	for _, log := range []string{"one", "two", "three", "four"} {
		session.publish(&testkube.TestWorkflowExecutionNotification{Log: log})
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	require.Len(t, session.replay, 3)
	require.Equal(t, notificationBytes("two", "three", "four"), session.replayBytes)
}

type fakeLiveLogStatsSource struct {
	kind  string
	stats liveLogStats
}

func (s fakeLiveLogStatsSource) liveLogKind() string        { return s.kind }
func (s fakeLiveLogStatsSource) liveLogStats() liveLogStats { return s.stats }

func TestLiveLogCollectorSumsSourcesByKind(t *testing.T) {
	collector := newLiveLogCollector()
	collector.add(fakeLiveLogStatsSource{kind: "workflow", stats: liveLogStats{activeSessions: 2, doneSessions: 1, subscribers: 3, replayBytes: 100}})
	collector.add(fakeLiveLogStatsSource{kind: "workflow", stats: liveLogStats{activeSessions: 1, subscribers: 1, replayBytes: 50}})
	collector.add(fakeLiveLogStatsSource{kind: "service", stats: liveLogStats{doneSessions: 4, replayBytes: 7}})
	gone := fakeLiveLogStatsSource{kind: "parallel", stats: liveLogStats{activeSessions: 9}}
	collector.add(gone)
	collector.remove(gone)

	expected := `
# HELP testkube_live_log_replay_bytes Approximate bytes held in live-log replay buffers
# TYPE testkube_live_log_replay_bytes gauge
testkube_live_log_replay_bytes{kind="service"} 7
testkube_live_log_replay_bytes{kind="workflow"} 150
# HELP testkube_live_log_sessions Current number of live-log streaming sessions by state
# TYPE testkube_live_log_sessions gauge
testkube_live_log_sessions{kind="service",state="active"} 0
testkube_live_log_sessions{kind="service",state="done"} 4
testkube_live_log_sessions{kind="workflow",state="active"} 3
testkube_live_log_sessions{kind="workflow",state="done"} 1
# HELP testkube_live_log_subscribers Current number of live-log stream subscribers
# TYPE testkube_live_log_subscribers gauge
testkube_live_log_subscribers{kind="service"} 0
testkube_live_log_subscribers{kind="workflow"} 4
`
	require.NoError(t, testutil.CollectAndCompare(collector, strings.NewReader(expected)))
}
