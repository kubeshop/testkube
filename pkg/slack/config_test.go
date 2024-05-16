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

	t.Run("allow all when selector, testname, testsuite, testworkflow is not specified", func(t *testing.T) {
		config := NewConfig(notificationConfig)
		assert.False(t, config.HasChannelsDefined())
		channels, needs := config.NeedsSending(&testkube.Event{Type_: testkube.EventStartTest})
		assert.True(t, needs)
		assert.Equal(t, 0, len(channels))
	})

	t.Run("allow only one event when only one event is specified", func(t *testing.T) {
		oldEvents := notificationConfig[0].Events
		notificationConfig[0].Events = []testkube.EventType{*testkube.EventStartTest}
		config := NewConfig(notificationConfig)
		channels, needs := config.NeedsSending(&testkube.Event{Type_: testkube.EventStartTest})
		assert.True(t, needs)
		assert.Equal(t, 0, len(channels))

		channels, needs = config.NeedsSending(&testkube.Event{Type_: testkube.EventEndTestSuccess})
		assert.False(t, needs)
		assert.Equal(t, 0, len(channels))
		notificationConfig[0].Events = oldEvents
	})

	t.Run("allow only one test when only one test is specified", func(t *testing.T) {
		oldTestNames := notificationConfig[0].TestNames
		notificationConfig[0].TestNames = []string{"test1"}
		config := NewConfig(notificationConfig)
		channels, needs := config.NeedsSending(&testkube.Event{TestExecution: &testkube.Execution{TestName: "test1"}, Type_: testkube.EventStartTest})
		assert.True(t, needs)
		assert.Equal(t, 0, len(channels))

		channels, needs = config.NeedsSending(&testkube.Event{TestExecution: &testkube.Execution{TestName: "test2"}, Type_: testkube.EventStartTest})
		assert.False(t, needs)
		assert.Equal(t, 0, len(channels))
		notificationConfig[0].TestNames = oldTestNames
	})

	t.Run("allow only one testsuite when only one testsuite is specified", func(t *testing.T) {
		oldTestSuiteNames := notificationConfig[0].TestSuiteNames
		notificationConfig[0].TestSuiteNames = []string{"testsuite1"}
		config := NewConfig(notificationConfig)
		testSuite := testkube.ObjectRef{Name: "testsuite1"}
		channels, needs := config.NeedsSending(&testkube.Event{TestSuiteExecution: &testkube.TestSuiteExecution{TestSuite: &testSuite}, Type_: testkube.EventStartTestSuite})
		assert.True(t, needs)
		assert.Equal(t, 0, len(channels))
		testSuite.Name = "testsuite2"
		channels, needs = config.NeedsSending(&testkube.Event{TestSuiteExecution: &testkube.TestSuiteExecution{TestSuite: &testSuite}, Type_: testkube.EventStartTestSuite})
		assert.False(t, needs)
		assert.Equal(t, 0, len(channels))
		notificationConfig[0].TestSuiteNames = oldTestSuiteNames
	})

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
		channels, needs := config.NeedsSending(&testkube.Event{Type_: testkube.EventStartTest})
		assert.True(t, needs)
		assert.Equal(t, 1, len(channels))
		assert.Equal(t, "channel1", channels[0])
		notificationConfig[0].ChannelID = oldChannelID
	})

	t.Run("allow only labels when selector is specified", func(t *testing.T) {
		oldSelector := notificationConfig[0].Selector
		notificationConfig[0].Selector = map[string]string{"label1": "value1"}
		config := NewConfig(notificationConfig)
		channels, needs := config.NeedsSending(&testkube.Event{TestExecution: &testkube.Execution{Labels: map[string]string{"label1": "value1"}}, Type_: testkube.EventStartTest})
		assert.True(t, needs)
		assert.Equal(t, 0, len(channels))
		channels, needs = config.NeedsSending(&testkube.Event{TestExecution: &testkube.Execution{Labels: map[string]string{"label1": "value2"}}, Type_: testkube.EventStartTest})
		assert.False(t, needs)
		assert.Equal(t, 0, len(channels))
		notificationConfig[0].Selector = oldSelector
	})

	t.Run("don't allow events when events do not match", func(t *testing.T) {
		oldEvents := notificationConfig[0].Events
		notificationConfig[0].Events = []testkube.EventType{*testkube.EventStartTest}
		config := NewConfig(notificationConfig)
		channels, needs := config.NeedsSending(&testkube.Event{Type_: testkube.EventEndTestSuccess})
		assert.False(t, needs)
		assert.Equal(t, 0, len(channels))
		notificationConfig[0].Events = oldEvents
	})

	t.Run("don't allow anything when everything is empty", func(t *testing.T) {
		config := NewConfig([]NotificationsConfig{})
		channels, needs := config.NeedsSending(&testkube.Event{Type_: testkube.EventStartTest})
		assert.False(t, needs)
		assert.Equal(t, 0, len(channels))
	})

	t.Run("allow notification when multiple configs, channel is specified and event is end-test-failed", func(t *testing.T) {
		var notificationConfigMultiple []NotificationsConfig
		err := json.Unmarshal([]byte(multipleConfigurationString), &notificationConfigMultiple)
		assert.Nil(t, err)
		config := NewConfig(notificationConfigMultiple)
		channels, needs := config.NeedsSending(&testkube.Event{Type_: testkube.EventEndTestFailed})
		assert.True(t, needs)
		assert.Equal(t, 1, len(channels))
		assert.Equal(t, "ChannelID2", channels[0])
	})
}
