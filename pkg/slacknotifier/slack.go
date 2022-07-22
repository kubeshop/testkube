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
				"text": "Execution {{ .ExecutionID }} of {{ .TestName }} status {{ .Status }}"
			}
		},
		{
			"type": "context",
			"elements": [
				{
					"type": "image",
					"image_url": "{{ if eq .Status "failed" }}https://raw.githubusercontent.com/kubeshop/testkube/d3380bc4bf4534ef1fb88cdce5d346dca8898986/assets/imageFailed.png{{ else if eq .Status "passed" }}https://raw.githubusercontent.com/kubeshop/testkube/d3380bc4bf4534ef1fb88cdce5d346dca8898986/assets/imagePassed.png{{ else }}https://raw.githubusercontent.com/kubeshop/testkube/d3380bc4bf4534ef1fb88cdce5d346dca8898986/assets/imagePending.png{{ end }}",
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
					"text": "*Labels*"
				},
				{
					"type": "plain_text",
					"text": "{{ .Namespace }} ",
					"emoji": true
				},
				{
					"type": "plain_text",
					"text": "{{ .Labels }} ",
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
	Labels      string
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

var (
	slackClient *slack.Client
	timestamps  map[string]string
)

func init() {
	timestamps = make(map[string]string)
	if token, ok := os.LookupEnv("SLACK_TOKEN"); ok {
		log.DefaultLogger.Info("initializing slack client", "SLACK_TOKEN", token)
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

	message, err := composeMessage(execution, eventType)
	if err != nil {
		return err
	}

	view := slack.Message{}
	err = json.Unmarshal(message, &view)
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
			prevTimestamp, ok := timestamps[execution.Name]
			var timestamp string

			if ok {
				_, timestamp, _, err = slackClient.UpdateMessage(channelID, prevTimestamp, slack.MsgOptionBlocks(view.Blocks.BlockSet...))
			} else {
				_, timestamp, err = slackClient.PostMessage(channelID, slack.MsgOptionBlocks(view.Blocks.BlockSet...))
			}

			if err != nil {
				log.DefaultLogger.Warnw("error while posting message to channel", "channelID", channelID, "error", err.Error())
				return err
			}

			if *eventType == testkube.END_TEST_WebhookEventType {
				delete(timestamps, execution.Name)
			} else {
				timestamps[execution.Name] = timestamp
			}
		} else {
			log.DefaultLogger.Warnw("Testkube bot is not added to any channel")
		}
	} else {
		log.DefaultLogger.Warnw("slack client is not initialised")
	}

	return nil
}

func composeMessage(execution testkube.Execution, eventType *testkube.WebhookEventType) ([]byte, error) {
	t, err := template.New("message").Parse(messageTemplate)
	if err != nil {
		log.DefaultLogger.Warnw("error while parsing slack template", "error", err.Error())
		return nil, err
	}

	args := messageArgs{
		ExecutionID: execution.Name,
		EventType:   string(*eventType),
		Namespace:   execution.TestNamespace,
		Labels:      testkube.MapToString(execution.Labels),
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

	log.DefaultLogger.Infow("Execution changed", "status", execution.ExecutionResult.Status)

	var message bytes.Buffer
	err = t.Execute(&message, args)
	if err != nil {
		log.DefaultLogger.Warnw("error while executing slack template", "error", err.Error())
		return nil, err
	}
	return message.Bytes(), nil
}
