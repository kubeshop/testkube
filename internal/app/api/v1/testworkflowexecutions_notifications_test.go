package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
)

func TestStreamableWorkflowNotificationsAppliesResumeCursorAndSequencing(t *testing.T) {
	watcher := executionworkertypes.NewNotificationsWatcher()
	go func() {
		watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "temporary", Temporary: true})
		watcher.Send(&testkube.TestWorkflowExecutionNotification{Log: "first"})
		watcher.Send(&testkube.TestWorkflowExecutionNotification{Result: &testkube.TestWorkflowResult{}})
		watcher.Close(nil)
	}()

	var notifications []testkube.TestWorkflowExecutionNotification
	for notification := range streamableWorkflowNotifications(watcher.Channel(), 1) {
		notifications = append(notifications, notification)
	}

	require.Len(t, notifications, 2)
	assert.Zero(t, notifications[0].SeqNo)
	assert.True(t, notifications[0].Temporary)
	assert.Equal(t, "log", notifications[0].EventType)
	assert.Equal(t, "temporary", notifications[0].Log)

	assert.Equal(t, int32(2), notifications[1].SeqNo)
	assert.Equal(t, "result", notifications[1].EventType)
	require.NotNil(t, notifications[1].Result)
}

func TestStreamableWorkflowNotificationsSendsHeartbeatWhileQuiet(t *testing.T) {
	done := make(chan struct{})
	defer close(done)
	source := make(chan *testkube.TestWorkflowExecutionNotification)

	notifications := streamableWorkflowNotificationsWithHeartbeat(done, source, 0, 5*time.Millisecond)

	select {
	case notification := <-notifications:
		assert.Equal(t, "heartbeat", notification.EventType)
		assert.Zero(t, notification.SeqNo)
		assert.Empty(t, notification.Log)
		assert.Nil(t, notification.Output)
		assert.Nil(t, notification.Result)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected heartbeat while workflow notification stream is quiet")
	}
}

func TestStreamableWorkflowNotificationsHeartbeatDoesNotAdvanceSeqNo(t *testing.T) {
	done := make(chan struct{})
	defer close(done)
	source := make(chan *testkube.TestWorkflowExecutionNotification, 1)

	notifications := streamableWorkflowNotificationsWithHeartbeat(done, source, 0, 5*time.Millisecond)

	select {
	case notification := <-notifications:
		require.Equal(t, "heartbeat", notification.EventType)
		assert.Zero(t, notification.SeqNo)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected initial heartbeat")
	}

	source <- &testkube.TestWorkflowExecutionNotification{Log: "after quiet period"}

	timeout := time.After(100 * time.Millisecond)
	for {
		select {
		case notification := <-notifications:
			if notification.EventType == "heartbeat" {
				continue
			}
			assert.Equal(t, "log", notification.EventType)
			assert.Equal(t, int32(1), notification.SeqNo)
			assert.Equal(t, "after quiet period", notification.Log)
			return
		case <-timeout:
			t.Fatal("expected durable log notification after heartbeat")
		}
	}
}

func TestWorkflowNotificationEventType(t *testing.T) {
	assert.Equal(t, "log", workflowNotificationEventType(testkube.TestWorkflowExecutionNotification{Log: "hello"}))
	assert.Equal(t, "result", workflowNotificationEventType(testkube.TestWorkflowExecutionNotification{Result: &testkube.TestWorkflowResult{}}))
	assert.Equal(t, "output", workflowNotificationEventType(testkube.TestWorkflowExecutionNotification{Output: &testkube.TestWorkflowOutput{Name: "out"}}))
	assert.Equal(t, "", workflowNotificationEventType(testkube.TestWorkflowExecutionNotification{}))
}

func TestWorkflowNotificationResumableIgnoresTemporaryNotifications(t *testing.T) {
	assert.False(t, workflowNotificationResumable(testkube.TestWorkflowExecutionNotification{Log: "temporary", Temporary: true}))
	assert.True(t, workflowNotificationResumable(testkube.TestWorkflowExecutionNotification{Log: "durable", Ts: time.Now()}))
}
