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
