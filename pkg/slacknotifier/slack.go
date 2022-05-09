package slacknotifier

import (
	"bytes"
	"encoding/json"
	"os"
	"text/template"

	"github.com/slack-go/slack"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// can be generated here https://app.slack.com/block-kit-builder
const messageTemplate string = `{
	"blocks": [
		{
			"type": "header",
			"text": {
				"type": "plain_text",
				"text": "Teskube activity",
				"emoji": true
			}
		},
		{
			"type": "section",
			"fields": [
				{
					"type": "mrkdwn",
					"text": "*Event Type:*\n{{ .EventType }}"
				}
				{{ if .Namespace }}
				,
				{
					"type": "mrkdwn",
					"text": "*Namespace:*\n{{ .Namespace }}"
				}
				{{ end }}
			]
		},
		{
			"type": "section",
			"fields": [
				{
					"type": "mrkdwn",
					"text": "*Test Name:*\n{{ .TestName }}"
				},
				{
					"type": "mrkdwn",
					"text": "*Test Type:*\n{{ .TestType }}"
				}
			]
		},
		{
			"type": "section",
			"fields": [
				{
					"type": "mrkdwn",
					"text": "*Status:*\n{{ .Status }}"
				}
			]
		},
		{
			"type": "section",
			"fields": [
				{
					"type": "mrkdwn",
					"text": "*Start Time:*\n{{ .StartTime }}"
				},
				{
					"type": "mrkdwn",
					"text": "*End Time:*\n{{ .EndTime }}"
				}
			]
		}
		{{ if .Duration }}
		,
		{
			"type": "section",
			"fields": [
				{
					"type": "mrkdwn",
					"text": "*Duration:*\n{{ .Duration }}"
				}
			]
		}
		{{ end }}
		{{ if .Output }}
		,
		{
			"type": "section",
			"fields": [
				{
					"type": "mrkdwn",
					"text": "*Output:*\n{{ .Output }}"
				}
			]
		}
		{{ end }}
	]
}`

type messageArgs struct {
	EventType string
	Namespace string
	TestName  string
	TestType  string
	Status    string
	StartTime string
	EndTime   string
	Duration  string
	Output    string
}

type client struct {
	SlackClient *slack.Client
	ChannelId   string
}

var c *client

func init() {
	if id, ok := os.LookupEnv("SLACK_CHANNEL_ID"); ok {
		c = &client{ChannelId: id}
		if token, ok := os.LookupEnv("SLACK_TOKEN"); ok {
			c.SlackClient = slack.New(token, slack.OptionDebug(true))
		}
	}
}

// SendMessage posts a message to the slack configured channel
func SendMessage(message string) error {
	if c != nil && c.SlackClient != nil {
		_, _, err := c.SlackClient.PostMessage(c.ChannelId, slack.MsgOptionText(message, false))
		if err != nil {
			return err
		}
	}
	return nil
}

// SendEvent composes an event message and sends it to slack
func SendEvent(eventType *testkube.WebhookEventType, execution testkube.Execution) error {

	t, err := template.New("message").Parse(messageTemplate)
	if err != nil {
		return err
	}

	args := messageArgs{
		EventType: string(*eventType),
		Namespace: execution.TestNamespace,
		TestName:  execution.TestName,
		TestType:  execution.TestType,
		Status:    string(*execution.ExecutionResult.Status),
		StartTime: execution.StartTime.String(),
		EndTime:   execution.EndTime.String(),
		Duration:  execution.Duration,
		Output:    execution.ExecutionResult.Output}

	var message bytes.Buffer
	err = t.Execute(&message, args)
	if err != nil {
		return err
	}

	view := slack.Message{}
	err = json.Unmarshal(message.Bytes(), &view)
	if err != nil {
		return err
	}
	if c != nil && c.SlackClient != nil {
		_, _, err := c.SlackClient.PostMessage(c.ChannelId, slack.MsgOptionBlocks(view.Blocks.BlockSet...))
		if err != nil {
			return err
		}
	}
	return nil
}
