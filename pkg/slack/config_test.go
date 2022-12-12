package slack

import (
	"encoding/json"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

const notificationConfigString = `[
    {
      "ChannelID": "",
      "selector": {},
      "testName": [],
      "testSuiteName": [],
      "events": [
        "start-test",
        "end-test-success",
        "end-test-failed",
        "end-test-aborted",
        "end-test-timeout",
        "start-testsuite",
        "end-testsuite-success",
        "end-testsuite-failed",
        "end-testsuite-aborted",
        "end-testsuite-timeout"
      ]
    }
  ]`

func TestParamsNilAssign(t *testing.T) {
	var notificationConfig []NotificationsConfig
	err := json.Unmarshal([]byte(notificationConfigString), &notificationConfig)

	assert.Nil(t, err)

	t.Run("allow all when no selector, testname, testsuite is not specified", func(t *testing.T) {
		config := NewConfig(notificationConfig)
		assert.False(t, config.HasChannelsDefined())
		channels, needs := config.NeedsSending(&testkube.Event{Type_: testkube.EventStartTest})
		assert.True(t, needs)
		assert.Equal(t, 0, len(channels))
	})

}
