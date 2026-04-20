package client

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestStreamToTestWorkflowExecutionNotificationsChannel_HandlesSSEFramesAndHeartbeats(t *testing.T) {
	stream := strings.NewReader(strings.Join([]string{
		": connected",
		"",
		`data: {"ref":"root","log":"hello"}`,
		"",
		": ping",
		"",
		`data: {"ref":"root","result":{"status":"passed","finishedAt":"2026-03-24T00:00:00Z"}}`,
		"",
	}, "\n"))

	notifications := make(chan testkube.TestWorkflowExecutionNotification, 4)
	StreamToTestWorkflowExecutionNotificationsChannel(stream, notifications)
	close(notifications)

	var received []testkube.TestWorkflowExecutionNotification
	for notification := range notifications {
		received = append(received, notification)
	}

	require.Len(t, received, 2)
	assert.Equal(t, "hello", received[0].Log)
	require.NotNil(t, received[1].Result)
	require.NotNil(t, received[1].Result.Status)
	assert.Equal(t, testkube.PASSED_TestWorkflowStatus, *received[1].Result.Status)
	assert.False(t, received[1].Result.FinishedAt.IsZero())
}

func TestStreamToTestWorkflowExecutionNotificationsChannel_HandlesSSEEventFieldsAndSeqNo(t *testing.T) {
	stream := strings.NewReader(strings.Join([]string{
		"id: 7",
		"event: heartbeat",
		`data: {"seqNo":7,"eventType":"heartbeat"}`,
		"",
		"id: 8",
		"event: log",
		`data: {"seqNo":8,"eventType":"log","ref":"root","log":"hello"}`,
		"",
	}, "\n"))

	notifications := make(chan testkube.TestWorkflowExecutionNotification, 4)
	StreamToTestWorkflowExecutionNotificationsChannel(stream, notifications)
	close(notifications)

	var received []testkube.TestWorkflowExecutionNotification
	for notification := range notifications {
		received = append(received, notification)
	}

	require.Len(t, received, 2)
	assert.Equal(t, int32(7), received[0].SeqNo)
	assert.Equal(t, "heartbeat", received[0].EventType)
	assert.Equal(t, int32(8), received[1].SeqNo)
	assert.Equal(t, "log", received[1].EventType)
	assert.Equal(t, "hello", received[1].Log)
}
