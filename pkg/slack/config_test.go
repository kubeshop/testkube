package slack

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const notificationConfigString = `[
    {
      "ChannelID": "",
      "selector": {},
      "testName": [],
      "testSuiteName": [],
	  "testWorkflowName": [],
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
        "end-testsuite-timeout",
        "start-testworkflow",
        "queue-testworkflow",
        "end-testworkflow-success",
        "end-testworkflow-failed",
        "end-testworkflow-aborted"
      ]
    }
  ]`

const multipleConfigurationString = `[
    {
      "ChannelID": "ChannelID1",
      "selector": {},
      "testName": [],
      "testSuiteName": [],
      "testWorkflowName": [],	  
      "events": [
        "start-test"
      ]
    },
    {
      "ChannelID": "ChannelID2",
      "selector": {},
      "testName": [],
      "testSuiteName": [],
	  "testWorkflowName": [],		  
      "events": [
        "end-test-failed"
      ]
    }
  ]`

func TestParamsNilAssign(t *testing.T) {
	var notificationConfig []NotificationsConfig
	err := json.Unmarshal([]byte(notificationConfigString), &notificationConfig)

	assert.Nil(t, err)

	t.Run("allow only one testworkflow when only one testworkflow is specified", func(t *testing.T) {
		oldTestWorkflowNames := notificationConfig[0].TestWorkflowNames
		notificationConfig[0].TestWorkflowNames = []string{"testworkflow1"}
		config := NewConfig(notificationConfig)
		channels, needs := config.NeedsSending(&testkube.Event{TestWorkflowExecution: &testkube.TestWorkflowExecution{Name: "testworkflow1"}, Type_: testkube.EventStartTestWorkflow})
		assert.True(t, needs)
		assert.Equal(t, 0, len(channels))
		channels, needs = config.NeedsSending(&testkube.Event{TestWorkflowExecution: &testkube.TestWorkflowExecution{Name: "testworkflow2"}, Type_: testkube.EventStartTestWorkflow})
		assert.False(t, needs)
		assert.Equal(t, 0, len(channels))
		notificationConfig[0].TestWorkflowNames = oldTestWorkflowNames
	})

	t.Run("allow only on one channel when only one channel is specified", func(t *testing.T) {
		oldChannelID := notificationConfig[0].ChannelID
		notificationConfig[0].ChannelID = "channel1"
		config := NewConfig(notificationConfig)
		channels, needs := config.NeedsSending(&testkube.Event{Type_: testkube.EventStartTestWorkflow})
		assert.True(t, needs)
		assert.Equal(t, 1, len(channels))
		assert.Equal(t, "channel1", channels[0])
		notificationConfig[0].ChannelID = oldChannelID
	})

	t.Run("allow only labels when selector is specified", func(t *testing.T) {
		oldSelector := notificationConfig[0].Selector
		notificationConfig[0].Selector = map[string]string{"label1": "value1"}
		config := NewConfig(notificationConfig)
		channels, needs := config.NeedsSending(&testkube.Event{TestWorkflowExecution: &testkube.TestWorkflowExecution{Workflow: &testkube.TestWorkflow{Labels: map[string]string{"label1": "value1"}}}, Type_: testkube.EventStartTestWorkflow})
		assert.True(t, needs)
		assert.Equal(t, 0, len(channels))
		channels, needs = config.NeedsSending(&testkube.Event{TestWorkflowExecution: &testkube.TestWorkflowExecution{Workflow: &testkube.TestWorkflow{Labels: map[string]string{"label1": "value2"}}}, Type_: testkube.EventStartTestWorkflow})
		assert.False(t, needs)
		assert.Equal(t, 0, len(channels))
		notificationConfig[0].Selector = oldSelector
	})

	t.Run("don't allow events when events do not match", func(t *testing.T) {
		oldEvents := notificationConfig[0].Events
		notificationConfig[0].Events = []testkube.EventType{*testkube.EventStartTestWorkflow}
		config := NewConfig(notificationConfig)
		channels, needs := config.NeedsSending(&testkube.Event{Type_: testkube.EventEndTestWorkflowSuccess})
		assert.False(t, needs)
		assert.Equal(t, 0, len(channels))
		notificationConfig[0].Events = oldEvents
	})

	t.Run("don't allow anything when everything is empty", func(t *testing.T) {
		config := NewConfig([]NotificationsConfig{})
		channels, needs := config.NeedsSending(&testkube.Event{Type_: testkube.EventStartTestWorkflow})
		assert.False(t, needs)
		assert.Equal(t, 0, len(channels))
	})

}
