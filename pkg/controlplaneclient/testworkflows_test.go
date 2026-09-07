package controlplaneclient

import (
	"context"
	"errors"
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
	session := newNotificationStreamSession(nil)
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
	session := newNotificationStreamSession(nil)

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
	session := newNotificationStreamSession(nil)
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

// newMetricsTestSessionManager builds a manager whose kind label is the test name.
// The metrics live on the shared default registry, so a unique kind keeps each
// test's readings apart from the other tests in the package.
func newMetricsTestSessionManager(t *testing.T, process func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher) *notificationStreamSessionManager[*cloud.TestWorkflowNotificationsRequest] {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return newNotificationStreamSessionManager(
		ctx,
		t.Name(),
		func(req *cloud.TestWorkflowNotificationsRequest) string { return req.ExecutionId },
		process,
	)
}

// blockingNotificationSource returns a process function whose watcher sends one
// log line and then stays open until the test finishes the execution id, with
// the error the source should end with.
func blockingNotificationSource(t *testing.T) (func(context.Context, *cloud.TestWorkflowNotificationsRequest) NotificationWatcher, func(executionID string, err error)) {
	t.Helper()

	type sourceEnd struct {
		err  error
		done chan struct{}
	}
	var mu sync.Mutex
	ends := make(map[string]*sourceEnd)
	process := func(_ context.Context, req *cloud.TestWorkflowNotificationsRequest) NotificationWatcher {
		end := &sourceEnd{done: make(chan struct{})}
		mu.Lock()
		ends[req.ExecutionId] = end
		mu.Unlock()
		watcher := channels.NewWatcher[*testkube.TestWorkflowExecutionNotification]()
		go func() {
			watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "line"})
			<-end.done
			watcher.Close(end.err)
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
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, end := range ends {
			select {
			case <-end.done:
			default:
				close(end.done)
			}
		}
	})
	return process, finish
}

func metricValue(t *testing.T, collector prometheus.Collector) float64 {
	t.Helper()
	return testutil.ToFloat64(collector)
}

// histogramSampleCount reads one label set's sample count. testutil.CollectAndCount
// on a curried vector counts every label set in the vector, so it cannot isolate a
// single kind on the shared registry.
func histogramSampleCount(t *testing.T, observer prometheus.Observer) uint64 {
	t.Helper()

	var metric dto.Metric
	require.NoError(t, observer.(prometheus.Metric).Write(&metric))
	return metric.GetHistogram().GetSampleCount()
}

func sessionReplayBytes(session *notificationStreamSession) int {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.replayBytes
}

func waitForNotificationSessionDone(t *testing.T, session *notificationStreamSession) {
	t.Helper()

	require.Eventually(t, func() bool {
		done, _ := session.status()
		return done
	}, 2*time.Second, time.Millisecond)
}

func TestNotificationStreamSessionManagerMetricsFollowSessionLifecycle(t *testing.T) {
	process, finish := blockingNotificationSource(t)
	manager := newMetricsTestSessionManager(t, process)
	manager.sessionIdleTTL = time.Minute
	kind := manager.kind

	// A fresh attach creates one active session with one subscriber.
	session, sub, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	assert.Equal(t, float64(1), metricValue(t, liveLogSessionsCreatedTotal.WithLabelValues(kind)))
	assert.Equal(t, float64(1), metricValue(t, liveLogSessions.WithLabelValues(kind, "active")))
	assert.Equal(t, float64(0), metricValue(t, liveLogSessions.WithLabelValues(kind, "done")))
	assert.Equal(t, float64(1), metricValue(t, liveLogSubscribers.WithLabelValues(kind)))

	// A second viewer that resumes the same stream shares the session.
	_, sub2, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 1})
	assert.Equal(t, float64(1), metricValue(t, liveLogSessionsCreatedTotal.WithLabelValues(kind)))
	assert.Equal(t, float64(2), metricValue(t, liveLogSubscribers.WithLabelValues(kind)))

	// The source ends: the session moves from active to done and records its duration.
	finish("exec-1", nil)
	waitForNotificationSessionDone(t, session)
	require.Eventually(t, func() bool {
		return metricValue(t, liveLogSessions.WithLabelValues(kind, "active")) == 0 &&
			metricValue(t, liveLogSessions.WithLabelValues(kind, "done")) == 1 &&
			histogramSampleCount(t, liveLogSourceDurationSeconds.WithLabelValues(kind)) == 1
	}, 2*time.Second, time.Millisecond)
	assert.Greater(t, metricValue(t, liveLogReplayBytes.WithLabelValues(kind)), float64(0))

	// Viewers leave: the subscriber gauge returns to zero.
	manager.detach(session, sub)
	manager.detach(session, sub2)
	assert.Equal(t, float64(0), metricValue(t, liveLogSubscribers.WithLabelValues(kind)))

	// The idle TTL passes: the sweep evicts the done session and frees its buffer.
	manager.sweepExpired(time.Now().Add(2 * manager.sessionIdleTTL))
	assert.Equal(t, float64(0), metricValue(t, liveLogSessions.WithLabelValues(kind, "done")))
	assert.Equal(t, float64(1), metricValue(t, liveLogSessionsEvictedTotal.WithLabelValues(kind, "ttl")))
	assert.Equal(t, float64(0), metricValue(t, liveLogReplayBytes.WithLabelValues(kind)))
}

func TestNotificationStreamSessionManagerReplayBytesGaugeTracksBufferedBytes(t *testing.T) {
	process, finish := blockingNotificationSource(t)
	manager := newMetricsTestSessionManager(t, process)
	manager.sessionIdleTTL = time.Minute
	kind := manager.kind
	gauge := func() float64 { return metricValue(t, liveLogReplayBytes.WithLabelValues(kind)) }

	// The viewers leave at once: this test only reads the replay buffer, and an
	// unread subscription would block publish once its channel fills.
	first, firstSub, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	manager.detach(first, firstSub)

	// The source's first line lands in the buffer before the test publishes more,
	// so every later reading compares a settled session against the gauge.
	require.Eventually(t, func() bool { return sessionReplayBytes(first) > 0 }, 2*time.Second, time.Millisecond)
	assert.Equal(t, float64(sessionReplayBytes(first)), gauge())

	// Every buffered notification adds its size to the gauge.
	first.publish(&testkube.TestWorkflowExecutionNotification{Log: "hello"})
	assert.Equal(t, float64(sessionReplayBytes(first)), gauge())

	// Once the buffer trims old events, the gauge follows the trimmed total.
	for i := 0; i < workflowNotificationReplayMaxEvents; i++ {
		first.publish(&testkube.TestWorkflowExecutionNotification{Log: "fill"})
	}
	first.mu.Lock()
	replayLen := len(first.replay)
	first.mu.Unlock()
	require.Equal(t, workflowNotificationReplayMaxEvents, replayLen)
	firstBytes := sessionReplayBytes(first)
	assert.Equal(t, float64(firstBytes), gauge())

	// Sessions of one kind share the gauge, so a second session adds to it.
	second, secondSub, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-2", StreamId: "stream-2"})
	manager.detach(second, secondSub)
	require.Eventually(t, func() bool { return sessionReplayBytes(second) > 0 }, 2*time.Second, time.Millisecond)
	second.publish(&testkube.TestWorkflowExecutionNotification{Log: "second"})
	secondBytes := sessionReplayBytes(second)
	assert.Equal(t, float64(firstBytes+secondBytes), gauge())

	// When the sweep evicts one session, the gauge drops by that session's bytes only.
	finish("exec-1", nil)
	waitForNotificationSessionDone(t, first)
	manager.sweepExpired(time.Now().Add(2 * manager.sessionIdleTTL))
	assert.Equal(t, float64(secondBytes), gauge())

	// When the sweep evicts the last session, the gauge returns to zero.
	finish("exec-2", nil)
	waitForNotificationSessionDone(t, second)
	manager.sweepExpired(time.Now().Add(2 * manager.sessionIdleTTL))
	assert.Equal(t, float64(0), gauge())
}

func TestNotificationStreamSessionManagerResumeMetricsCountResult(t *testing.T) {
	tests := []struct {
		name        string
		prepare     func(t *testing.T, manager *notificationStreamSessionManager[*cloud.TestWorkflowNotificationsRequest])
		req         *cloud.TestWorkflowNotificationsRequest
		available   float64
		unavailable float64
	}{
		{
			name: "resume inside the replay buffer is available",
			prepare: func(t *testing.T, manager *notificationStreamSessionManager[*cloud.TestWorkflowNotificationsRequest]) {
				session, sub, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
				t.Cleanup(func() { manager.detach(session, sub) })
				session.publish(&testkube.TestWorkflowExecutionNotification{Log: "two"})
			},
			req:         &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 1},
			available:   1,
			unavailable: 0,
		},
		{
			name:        "resume without a session is unavailable",
			prepare:     func(*testing.T, *notificationStreamSessionManager[*cloud.TestWorkflowNotificationsRequest]) {},
			req:         &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 5},
			available:   0,
			unavailable: 1,
		},
		{
			name:        "start from zero is not a resume",
			prepare:     func(*testing.T, *notificationStreamSessionManager[*cloud.TestWorkflowNotificationsRequest]) {},
			req:         &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"},
			available:   0,
			unavailable: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			process, _ := blockingNotificationSource(t)
			manager := newMetricsTestSessionManager(t, process)
			kind := manager.kind
			tt.prepare(t, manager)

			session, sub, _, _, _, _ := manager.attach(tt.req)
			t.Cleanup(func() { manager.detach(session, sub) })

			assert.Equal(t, tt.available, metricValue(t, liveLogResumeTotal.WithLabelValues(kind, "available")))
			assert.Equal(t, tt.unavailable, metricValue(t, liveLogResumeTotal.WithLabelValues(kind, "unavailable")))
		})
	}
}

func TestNotificationStreamSessionManagerEvictionMetricsCountReason(t *testing.T) {
	tests := []struct {
		name      string
		sourceErr error
		evict     func(t *testing.T, manager *notificationStreamSessionManager[*cloud.TestWorkflowNotificationsRequest])
		reason    string
	}{
		{
			name:      "idle TTL sweep evicts a done session",
			sourceErr: nil,
			evict: func(t *testing.T, manager *notificationStreamSessionManager[*cloud.TestWorkflowNotificationsRequest]) {
				manager.sweepExpired(time.Now().Add(2 * manager.sessionIdleTTL))
			},
			reason: "ttl",
		},
		{
			name:      "resume after a failed source evicts the errored session",
			sourceErr: errors.New("source failed"),
			evict: func(t *testing.T, manager *notificationStreamSessionManager[*cloud.TestWorkflowNotificationsRequest]) {
				session, sub, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 1})
				t.Cleanup(func() { manager.detach(session, sub) })
			},
			reason: "error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			process, finish := blockingNotificationSource(t)
			manager := newMetricsTestSessionManager(t, process)
			manager.sessionIdleTTL = time.Minute
			kind := manager.kind

			session, sub, _, _, _, _ := manager.attach(&cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
			finish("exec-1", tt.sourceErr)
			waitForNotificationSessionDone(t, session)
			manager.detach(session, sub)
			require.Eventually(t, func() bool {
				return metricValue(t, liveLogSessions.WithLabelValues(kind, "done")) == 1
			}, 2*time.Second, time.Millisecond)

			tt.evict(t, manager)

			assert.Equal(t, float64(1), metricValue(t, liveLogSessionsEvictedTotal.WithLabelValues(kind, tt.reason)))
			assert.Equal(t, float64(0), metricValue(t, liveLogSessions.WithLabelValues(kind, "done")))
			manager.mu.Lock()
			current := manager.sessions["exec-1:stream-1"]
			manager.mu.Unlock()
			assert.NotSame(t, session, current, "evicted session must leave the manager")
		})
	}
}
