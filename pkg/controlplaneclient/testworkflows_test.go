package controlplaneclient

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

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

	session, sub, replay, available, _, done := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	require.True(t, available)
	require.False(t, done)
	require.Empty(t, replay)

	firstPass := collectNotificationSubscriptionSeqNos(t, sub)
	session.unsubscribe(sub)

	assert.Equal(t, []uint32{1, 2, 3}, firstPass)

	session, sub, replay, available, lastSeqNo, done := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 1})
	require.True(t, available)
	require.True(t, done)
	require.Equal(t, uint32(3), lastSeqNo)
	require.Len(t, replay, 2)
	assert.Equal(t, []uint32{2, 3}, []uint32{replay[0].seqNo, replay[1].seqNo})
	session.unsubscribe(sub)
}

func TestSendNotificationResponseReturnsContextErrorWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	responses := make(chan string)
	err := sendNotificationResponse(ctx, responses, "response")

	require.ErrorIs(t, err, context.Canceled)
}

func TestNotificationStreamSessionPublishDoesNotHoldLockForSlowSubscriber(t *testing.T) {
	session := newNotificationStreamSession()
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
	session := newNotificationStreamSession()

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
	session := newNotificationStreamSession()
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

	session1, sub1, _, _, _, _ := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	waitForNotificationSubscriptionDone(t, sub1)
	session1.unsubscribe(sub1)

	session2, sub2, replay, available, _, _ := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
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

	session1, sub1, _, _, _, _ := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	firstPass := collectNotificationSubscriptionSeqNos(t, sub1)
	session1.unsubscribe(sub1)
	require.NotEmpty(t, firstPass)

	session2, sub2, replay, available, lastSeqNo, done := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: firstPass[len(firstPass)-1]})
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

	session, sub, replay, available, lastSeqNo, done := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{
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

	session, sub, replay, available, lastSeqNo, done := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1", ResumeAfterSeqNo: 7})
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

	session1, sub1, _, available1, _, done1 := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	defer session1.unsubscribe(sub1)
	session2, sub2, replay2, available2, _, done2 := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-2"})
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

	session, sub, _, _, _, _ := manager.attach(ctx, &cloud.TestWorkflowNotificationsRequest{ExecutionId: "exec-1", StreamId: "stream-1"})
	close(release)
	waitForNotificationSubscriptionDone(t, sub)
	session.unsubscribe(sub)

	require.Eventually(t, func() bool {
		manager.mu.Lock()
		defer manager.mu.Unlock()
		return len(manager.sessions) == 0
	}, time.Second, time.Millisecond)
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
