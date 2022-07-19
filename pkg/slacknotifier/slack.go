package slacknotifier

import (
	"bytes"
	"encoding/json"
	"os"
	"text/template"

	"github.com/slack-go/slack"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
)

// can be generated here https://app.slack.com/block-kit-builder
const messageTemplate string = `{
	"blocks": [
		{
			"type": "section",
			"text": {
				"type": "plain_text",
				"emoji": true,
				"text": "Execution {{ .ExecutionID }} of {{ .TestName }} reports status {{ .Status }}"
			}
		},
		{
			"type": "context",
			"elements": [
				{
					"type": "image",
					"image_url": "{{ if eq .Status "failed" }}https://icon-library.com/images/error-image-icon/error-image-icon-23.jpg{{ else }}https://icon-library.com/images/green-tick-icon/green-tick-icon-6.jpg{{ end }}",
					"alt_text": "notifications warning icon"
				}
				{{ if (gt .TotalSteps 0 )}}
				,
				{
					"type": "mrkdwn",
					"text": "*   {{ .FailedSteps }}/{{ .TotalSteps }} STEPS FAILED*"
				}
				{{ end }}
			]
		},
		{
			"type": "divider"
		},
		{
			"type": "section",
			"fields": [
				{
					"type": "mrkdwn",
					"text": "*Test Name*"
				},
				{
					"type": "mrkdwn",
					"text": "*Type*"
				},
				{
					"type": "plain_text",
					"text": "{{ .TestName }}",
					"emoji": true
				},
				{
					"type": "plain_text",
					"text": "{{ .TestType }}",
					"emoji": true
				}
			]
		},
		{{ if .Namespace}}
		{
			"type": "section",
			"fields": [
				{
					"type": "mrkdwn",
					"text": "*Namespace*"
				},
				{
					"type": "mrkdwn",
					"text": " "
				},
				{
					"type": "plain_text",
					"text": "{{ .Namespace }}",
					"emoji": true
				}
			]
		},
		{{ end }}
		{
			"type": "section",
			"fields": [
				{
					"type": "mrkdwn",
					"text": "*Start Time*"
				},
				{
					"type": "mrkdwn",
					"text": "*End Time*"
				},
				{
					"type": "plain_text",
					"text": "{{ .StartTime }}",
					"emoji": true
				},
				{
					"type": "plain_text",
					"text": "{{ .EndTime }}",
					"emoji": true
				}
			]
		},
		{{ if .Duration }}
		{
			"type": "section",
			"fields": [
				{
					"type": "mrkdwn",
					"text": "*Duration*"
				},
				{
					"type": "mrkdwn",
					"text": " "
				},
				{
					"type": "plain_text",
					"text": "{{ .Duration }}",
					"emoji": true
				}
			]
		},
		{{ end }}
		{
			"type": "divider"
		},
		{
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": "*Test Execution Results*"
			}
		},
		{
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": "{{ .BackTick }}kubectl testkube get execution {{ .ExecutionID }} {{ .BackTick }}\n"
			}
		},
		{
			"type": "divider"
		}
	]
}`

type messageArgs struct {
	ExecutionID string
	EventType   string
	Namespace   string
	TestName    string
	TestType    string
	Status      string
	FailedSteps int
	TotalSteps  int
	StartTime   string
	EndTime     string
	Duration    string
	BackTick    string
}

var slackClient *slack.Client

func init() {
	if token, ok := os.LookupEnv("SLACK_TOKEN"); ok {
		log.DefaultLogger.Info("initializing slack client", "SLACK_TOKEN", token[len(token)-5:])
		slackClient = slack.New(token, slack.OptionDebug(true))
	} else {
		log.DefaultLogger.Warn("SLACK_TOKEN is not set")
	}
}

// SendMessage posts a message to the slack configured channel
func SendMessage(channelID string, message string) error {
	if slackClient != nil {
		_, _, err := slackClient.PostMessage(channelID, slack.MsgOptionText(message, false))
		if err != nil {
			log.DefaultLogger.Warnw("error while posting message to channel", "channelID", channelID, "error", err.Error())
			return err
		}
	} else {
		log.DefaultLogger.Warnw("slack client is not initialised")
	}
	return nil
}

// SendEvent composes an event message and sends it to slack
func SendEvent(eventType *testkube.WebhookEventType, execution testkube.Execution) error {

	t, err := template.New("message").Parse(messageTemplate)
	if err != nil {
		log.DefaultLogger.Warnw("error while parsing slack template", "error", err.Error())
		return err
	}

	args := messageArgs{
		ExecutionID: execution.Id,
		EventType:   string(*eventType),
		Namespace:   execution.TestNamespace,
		TestName:    execution.TestName,
		TestType:    execution.TestType,
		Status:      string(*execution.ExecutionResult.Status),
		StartTime:   execution.StartTime.String(),
		EndTime:     execution.EndTime.String(),
		Duration:    execution.Duration,
		TotalSteps:  len(execution.ExecutionResult.Steps),
		FailedSteps: execution.ExecutionResult.GetFailedStepsCount(),
		BackTick:    "`",
	}

	var message bytes.Buffer
	err = t.Execute(&message, args)
	if err != nil {
		log.DefaultLogger.Warnw("error while executing slack template", "error", err.Error())
		return err
	}

	view := slack.Message{}
	err = json.Unmarshal(message.Bytes(), &view)
	if err != nil {
		log.DefaultLogger.Warnw("error while creating slack specific message", "error", err.Error())
		return err
	}

	if slackClient != nil {
		channels, _, err := slackClient.GetConversationsForUser(&slack.GetConversationsForUserParameters{})
		if err != nil {
			log.DefaultLogger.Warnw("error while getting bot channels", "error", err.Error())
			return err
		}

		if len(channels) > 0 {
			channelID := channels[0].GroupConversation.ID

			_, _, err := slackClient.PostMessage(channelID, slack.MsgOptionBlocks(view.Blocks.BlockSet...))
			if err != nil {
				log.DefaultLogger.Warnw("error while posting message to channel", "channelID", channelID, "error", err.Error())
				return err
			}
		} else {
			log.DefaultLogger.Warnw("Testkube bot is not added to any channel")
		}
	} else {
		log.DefaultLogger.Warnw("slack client is not initialised")
	}

	return nil
}
