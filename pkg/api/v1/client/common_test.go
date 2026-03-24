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
