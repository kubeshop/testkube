package slack

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type NotificationsConfig struct {
	ChannelID         string               `json:"channelID,omitempty"`
	Selector          map[string]string    `json:"selector,omitempty"`
	TestNames         []string             `json:"testName,omitempty"`
	TestSuiteNames    []string             `json:"testSuiteName,omitempty"`
	TestWorkflowNames []string             `json:"testWorkflowName,omitempty"`
	Events            []testkube.EventType `json:"events,omitempty"`
}

type NotificationsConfigSet struct {
	ChannelID         string
	Selector          map[string]string
	TestNames         *set[string]
	TestSuiteNames    *set[string]
	TestWorkflowNames *set[string]
	Events            *set[testkube.EventType]
}

type Config struct {
	NotificationsConfigSet []NotificationsConfigSet
	hasChannelsDefined     bool
}

func NewConfigSet(config NotificationsConfig) NotificationsConfigSet {
	return NotificationsConfigSet{
		ChannelID:         config.ChannelID,
		Selector:          config.Selector,
		TestNames:         NewSetFromArray(config.TestNames),
		TestSuiteNames:    NewSetFromArray(config.TestSuiteNames),
		TestWorkflowNames: NewSetFromArray(config.TestWorkflowNames),
		Events:            NewSetFromArray(config.Events),
	}
}

func NewConfig(config []NotificationsConfig) *Config {
	cf := Config{}
	cf.NotificationsConfigSet = make([]NotificationsConfigSet, len(config))
	for i, c := range config {
		if c.ChannelID != "" {
			cf.hasChannelsDefined = true
		}
		cf.NotificationsConfigSet[i] = NewConfigSet(c)
	}
	return &cf
}

func (c *Config) HasChannelsDefined() bool {
	return c.hasChannelsDefined
}

func (c *Config) NeedsSending(event *testkube.Event) ([]string, bool) {
	channels := []string{}
	var labels map[string]string
	if event.TestExecution != nil {
		labels = event.TestExecution.Labels
	}
	if event.TestSuiteExecution != nil {
		labels = event.TestSuiteExecution.Labels
	}
	if event.TestWorkflowExecution != nil && event.TestWorkflowExecution.Workflow != nil {
		labels = event.TestWorkflowExecution.Workflow.Labels
	}

	testName := ""
	if event.TestExecution != nil {
		testName = event.TestExecution.TestName
	}
	testSuiteName := ""
	if event.TestSuiteExecution != nil {
		testSuiteName = event.TestSuiteExecution.TestSuite.Name
	}
	testWorkflowName := ""
	if event.TestWorkflowExecution != nil {
		testWorkflowName = event.TestWorkflowExecution.Name
	}

	for _, config := range c.NotificationsConfigSet {

		hasMatch := false
		if config.Events != nil &&
			config.Events.Contains(event.Type()) &&
			config.TestSuiteNames.IsEmpty() &&
			config.TestNames.IsEmpty() &&
			config.TestWorkflowNames.IsEmpty() &&
			len(config.Selector) == 0 {
			hasMatch = true
		}

		if config.Selector != nil {
			for k, v := range config.Selector {
				if labels != nil && labels[k] == v {
					hasMatch = true
				}
			}
		}
		if config.TestNames != nil {
			if config.TestNames.Contains(testName) {
				hasMatch = true
			}
		}
		if config.TestSuiteNames != nil {
			if config.TestSuiteNames.Contains(testSuiteName) {
				hasMatch = true
			}
		}
		if config.TestWorkflowNames != nil {
			if config.TestWorkflowNames.Contains(testWorkflowName) {
				hasMatch = true
			}
		}
		if config.Events != nil && config.Events.Contains(event.Type()) && hasMatch {
			if config.ChannelID != "" {
				channels = append(channels, config.ChannelID)
			} else {
				return channels, true
			}

		}
	}
	if len(channels) > 0 {
		return channels, true
	}
	return channels, false
}
