package slack

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/slack-go/slack"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils"
	"github.com/kubeshop/testkube/pkg/utils/text"
)

type MessageArgs struct {
	ExecutionID   string
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
	ClusterName   string
	DashboardURI  string
	Envs          map[string]string
}

type Notifier struct {
	client          *slack.Client
	timestamps      map[string]string
	Ready           bool
	messageTemplate string
	clusterName     string
	dashboardURI    string
	config          *Config
	envs            map[string]string
}

func NewNotifier(template, clusterName, dashboardURI string, config []NotificationsConfig, envs map[string]string) *Notifier {
	notifier := Notifier{messageTemplate: template, clusterName: clusterName, dashboardURI: dashboardURI,
		config: NewConfig(config), envs: envs}
	notifier.timestamps = make(map[string]string)
	if token, ok := os.LookupEnv("SLACK_TOKEN"); ok && token != "" {
		log.DefaultLogger.Infow("initializing slack client", "SLACK_TOKEN", text.Obfuscate(token))
		notifier.client = slack.New(token, slack.OptionDebug(true))
		notifier.Ready = true
	} else {
		log.DefaultLogger.Warn("SLACK_TOKEN is not set")
	}
	return &notifier
}

// SendMessage posts a message to the slack configured channel
func (s *Notifier) SendMessage(channelID string, message string) error {
	if s.client != nil {
		_, _, err := s.client.PostMessage(channelID, slack.MsgOptionText(message, false))
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
func (s *Notifier) SendEvent(event *testkube.Event) error {

	message, name, err := s.composeMessage(event)
	if err != nil {
		return err
	}

	if s.client != nil {

		log.DefaultLogger.Debugw("sending event to slack", "event", event)
		channels, err := s.getChannels(event)
		if err != nil {
			return err
		}
		log.DefaultLogger.Infow("channels to send event to", "channels", channels)

		for _, channelID := range channels {
			prevTimestamp, ok := s.timestamps[name]
			var timestamp string

			if ok {
				_, timestamp, _, err = s.client.UpdateMessage(channelID, prevTimestamp, slack.MsgOptionBlocks(message.Blocks.BlockSet...))
			}

			if !ok || err != nil {
				_, timestamp, err = s.client.PostMessage(channelID, slack.MsgOptionBlocks(message.Blocks.BlockSet...))
			}

			if err != nil {
				log.DefaultLogger.Warnw("error while posting message to channel",
					"channelID", channelID,
					"error", err.Error(),
					"slackMessageOptions", slack.MsgOptionBlocks(message.Blocks.BlockSet...))
				return err
			}

			if event.IsSuccess() {
				delete(s.timestamps, name)
			} else {
				s.timestamps[name] = timestamp
			}
		}
	} else {
		log.DefaultLogger.Warnw("slack client is not initialised")
	}

	return nil
}

func (s *Notifier) getChannels(event *testkube.Event) ([]string, error) {
	result := []string{}
	if !s.config.HasChannelsDefined() {
		channels, _, err := s.client.GetConversationsForUser(&slack.GetConversationsForUserParameters{})
		if err != nil {
			log.DefaultLogger.Warnw("error while getting bot channels", "error", err.Error())
			return nil, err
		}
		_, needsSending := s.config.NeedsSending(event)
		if len(channels) > 0 && needsSending {
			result = append(result, channels[0].GroupConversation.ID)
			return result, nil
		}
	} else {
		channels, needsSending := s.config.NeedsSending(event)
		if needsSending {
			return channels, nil
		}
	}
	return nil, nil
}

func (s *Notifier) composeMessage(event *testkube.Event) (view *slack.Message, name string, err error) {
	var message []byte
	if event.TestExecution != nil {
		message, err = s.composeTestMessage(event.TestExecution, event.Type())
		name = event.TestExecution.Name
	} else if event.TestSuiteExecution != nil {
		message, err = s.composeTestsuiteMessage(event.TestSuiteExecution, event.Type())
		name = event.TestSuiteExecution.Name
	} else if event.TestWorkflowExecution != nil {
		message, err = s.composeTestWorkflowMessage(event.TestWorkflowExecution, event.Type())
		name = event.TestWorkflowExecution.Name
	} else {
		log.DefaultLogger.Warnw("event type is not handled by Slack notifier", "event", event)
		return nil, "", nil
	}

	if err != nil {
		return nil, "", err
	}
	view = &slack.Message{}
	err = json.Unmarshal(message, view)
	if err != nil {
		log.DefaultLogger.Warnw("error while creating slack specific message", "error", err.Error(), "message", string(message))
		return nil, "", err
	}

	return view, name, nil
}

func (s *Notifier) composeTestsuiteMessage(execution *testkube.TestSuiteExecution, eventType testkube.EventType) ([]byte, error) {
	t, err := utils.NewTemplate("message").Parse(s.messageTemplate)
	if err != nil {
		log.DefaultLogger.Warnw("error while parsing slack template", "error", err.Error())
		return nil, err
	}

	args := MessageArgs{
		ExecutionID:   execution.Id,
		ExecutionName: execution.Name,
		EventType:     string(eventType),
		Namespace:     execution.TestSuite.Namespace,
		Labels:        testkube.MapToString(execution.Labels),
		TestName:      execution.TestSuite.Name,
		TestType:      "Test Suite",
		Status:        string(*execution.Status),
		StartTime:     execution.StartTime.String(),
		EndTime:       execution.EndTime.String(),
		Duration:      execution.Duration,
		TotalSteps:    len(execution.ExecuteStepResults),
		FailedSteps:   execution.FailedStepsCount(),
		ClusterName:   s.clusterName,
		DashboardURI:  s.dashboardURI,
		Envs:          s.envs,
	}

	log.DefaultLogger.Infow("Execution changed", "status", execution.Status)

	var message bytes.Buffer
	err = t.Execute(&message, args)
	if err != nil {
		log.DefaultLogger.Warnw("error while executing slack template", "error", err.Error(), "template", s.messageTemplate, "args", args)
		return nil, err
	}
	return message.Bytes(), nil
}

func (s *Notifier) composeTestWorkflowMessage(execution *testkube.TestWorkflowExecution, eventType testkube.EventType) ([]byte, error) {
	t, err := utils.NewTemplate("message").Parse(s.messageTemplate)
	if err != nil {
		log.DefaultLogger.Warnw("error while parsing slack template", "error", err.Error())
		return nil, err
	}

	var name, namespace string
	var labels map[string]string
	if execution.Workflow != nil {
		name = execution.Workflow.Name
		namespace = execution.Workflow.Namespace
		labels = execution.Workflow.Labels
	}

	var status, startTime, endTime, duration string
	var totalSteps, failedSteps int
	if execution.Result != nil {
		status = string(*execution.Result.Status)
		startTime = execution.Result.StartedAt.String()
		endTime = execution.Result.FinishedAt.String()
		duration = execution.Result.Duration
		totalSteps = len(execution.Result.Steps)
		for _, step := range execution.Result.Steps {
			if step.Status != nil && *step.Status == testkube.FAILED_TestWorkflowStepStatus {
				failedSteps++
			}
		}
	}

	args := MessageArgs{
		ExecutionID:   execution.Id,
		ExecutionName: execution.Name,
		EventType:     string(eventType),
		Namespace:     namespace,
		Labels:        testkube.MapToString(labels),
		TestName:      name,
		TestType:      "Test Workflow",
		Status:        status,
		StartTime:     startTime,
		EndTime:       endTime,
		Duration:      duration,
		TotalSteps:    totalSteps,
		FailedSteps:   failedSteps,
		ClusterName:   s.clusterName,
		DashboardURI:  s.dashboardURI,
		Envs:          s.envs,
	}

	log.DefaultLogger.Infow("Execution changed", "status", status)

	var message bytes.Buffer
	err = t.Execute(&message, args)
	if err != nil {
		log.DefaultLogger.Warnw("error while executing slack template", "error", err.Error(), "template", s.messageTemplate, "args", args)
		return nil, err
	}
	return message.Bytes(), nil
}

func (s *Notifier) composeTestMessage(execution *testkube.Execution, eventType testkube.EventType) ([]byte, error) {
	t, err := utils.NewTemplate("message").Parse(s.messageTemplate)
	if err != nil {
		log.DefaultLogger.Warnw("error while parsing slack template", "error", err.Error(), "template", s.messageTemplate)
		return nil, err
	}

	args := MessageArgs{
		ExecutionID:   execution.Id,
		ExecutionName: execution.Name,
		EventType:     string(eventType),
		Namespace:     execution.TestNamespace,
		Labels:        testkube.MapToString(execution.Labels),
		TestName:      execution.TestName,
		TestType:      execution.TestType,
		Status:        string(testkube.QUEUED_ExecutionStatus),
		StartTime:     execution.StartTime.String(),
		EndTime:       execution.EndTime.String(),
		Duration:      execution.Duration,
		TotalSteps:    0,
		FailedSteps:   0,
		ClusterName:   s.clusterName,
		DashboardURI:  s.dashboardURI,
		Envs:          s.envs,
	}

	if execution.ExecutionResult != nil {
		if execution.ExecutionResult.Status != nil {
			args.Status = string(*execution.ExecutionResult.Status)
		}
		args.TotalSteps = len(execution.ExecutionResult.Steps)
		args.FailedSteps = execution.ExecutionResult.FailedStepsCount()
	}

	log.DefaultLogger.Infow("Execution changed", "status", execution.ExecutionResult.Status)

	var message bytes.Buffer
	err = t.Execute(&message, args)
	if err != nil {
		log.DefaultLogger.Warnw("error while executing slack template", "error", err.Error(), "template", s.messageTemplate, "args", args)
		return nil, err
	}
	return message.Bytes(), nil
}
