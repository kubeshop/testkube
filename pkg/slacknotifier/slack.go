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

type messageArgs struct {
	ExecutionName string
	EventType     string
	Namespace     string
	Labels        string
	TestName      string
	TestType      string
	Status        string
	FailedSteps   int
	TotalSteps    int
	StartTime     string
	EndTime       string
	Duration      string
}

type SlackNotifier struct {
	slackClient     *slack.Client
	timestamps      map[string]string
	Ready           bool
	messageTemplate string
}

func NewSlackNotifier(template string) *SlackNotifier {
	slackNotifier := SlackNotifier{messageTemplate: template}
	slackNotifier.timestamps = make(map[string]string)
	if token, ok := os.LookupEnv("SLACK_TOKEN"); ok {
		log.DefaultLogger.Info("initializing slack client", "SLACK_TOKEN", token)
		slackNotifier.slackClient = slack.New(token, slack.OptionDebug(true))
		slackNotifier.Ready = true
	} else {
		log.DefaultLogger.Warn("SLACK_TOKEN is not set")
	}
	return &slackNotifier
}

// SendMessage posts a message to the slack configured channel
func (s *SlackNotifier) SendMessage(channelID string, message string) error {
	if s.slackClient != nil {
		_, _, err := s.slackClient.PostMessage(channelID, slack.MsgOptionText(message, false))
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
func (s *SlackNotifier) SendEvent(event testkube.Event) error {
	var (
		message []byte
		err     error
		name    string
	)

	if event.TestExecution != nil {
		message, err = s.composeTestMessage(*event.TestExecution, event.Type())
		name = event.TestExecution.Name
	} else if event.TestSuiteExecution != nil {
		message, err = s.composeTestsuiteMessage(*event.TestSuiteExecution, event.Type())
		name = event.TestSuiteExecution.Name
	} else {
		log.DefaultLogger.Warnw("event type is not handled by Slack notifier", "event", event)
		return nil
	}

	if err != nil {
		return err
	}

	view := slack.Message{}
	err = json.Unmarshal(message, &view)
	if err != nil {
		log.DefaultLogger.Warnw("error while creating slack specific message", "error", err.Error())
		return err
	}

	if s.slackClient != nil {
		channels, _, err := s.slackClient.GetConversationsForUser(&slack.GetConversationsForUserParameters{})
		if err != nil {
			log.DefaultLogger.Warnw("error while getting bot channels", "error", err.Error())
			return err
		}

		if len(channels) > 0 {
			channelID := channels[0].GroupConversation.ID
			prevTimestamp, ok := s.timestamps[name]
			var timestamp string

			if ok {
				_, timestamp, _, err = s.slackClient.UpdateMessage(channelID, prevTimestamp, slack.MsgOptionBlocks(view.Blocks.BlockSet...))
			} else {
				_, timestamp, err = s.slackClient.PostMessage(channelID, slack.MsgOptionBlocks(view.Blocks.BlockSet...))
			}

			if err != nil {
				log.DefaultLogger.Warnw("error while posting message to channel", "channelID", channelID, "error", err.Error())
				return err
			}

			if event.IsSuccess() {
				delete(s.timestamps, name)
			} else {
				s.timestamps[name] = timestamp
			}
		} else {
			log.DefaultLogger.Warnw("Testkube bot is not added to any channel")
		}
	} else {
		log.DefaultLogger.Warnw("slack client is not initialised")
	}

	return nil
}

func (s *SlackNotifier) composeTestsuiteMessage(execution testkube.TestSuiteExecution, eventType testkube.EventType) ([]byte, error) {
	t, err := template.New("message").Parse(s.messageTemplate)
	if err != nil {
		log.DefaultLogger.Warnw("error while parsing slack template", "error", err.Error())
		return nil, err
	}

	args := messageArgs{
		ExecutionName: execution.Name,
		EventType:     string(eventType),
		Namespace:     execution.TestSuite.Namespace,
		Labels:        testkube.MapToString(execution.Labels),
		TestName:      execution.TestSuite.Name,
		Status:        string(*execution.Status),
		StartTime:     execution.StartTime.String(),
		EndTime:       execution.EndTime.String(),
		Duration:      execution.Duration,
		TotalSteps:    len(execution.StepResults),
		FailedSteps:   execution.FailedStepsCount(),
	}

	log.DefaultLogger.Infow("Execution changed", "status", execution.Status)

	var message bytes.Buffer
	err = t.Execute(&message, args)
	if err != nil {
		log.DefaultLogger.Warnw("error while executing slack template", "error", err.Error())
		return nil, err
	}
	return message.Bytes(), nil
}

func (s *SlackNotifier) composeTestMessage(execution testkube.Execution, eventType testkube.EventType) ([]byte, error) {
	t, err := template.New("message").Parse(s.messageTemplate)
	if err != nil {
		log.DefaultLogger.Warnw("error while parsing slack template", "error", err.Error())
		return nil, err
	}

	args := messageArgs{
		ExecutionName: execution.Name,
		EventType:     string(eventType),
		Namespace:     execution.TestNamespace,
		Labels:        testkube.MapToString(execution.Labels),
		TestName:      execution.TestName,
		TestType:      execution.TestType,
		Status:        string(*execution.ExecutionResult.Status),
		StartTime:     execution.StartTime.String(),
		EndTime:       execution.EndTime.String(),
		Duration:      execution.Duration,
		TotalSteps:    len(execution.ExecutionResult.Steps),
		FailedSteps:   execution.ExecutionResult.FailedStepsCount(),
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
