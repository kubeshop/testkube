package slack

import (
	"encoding/json"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/slack"
	"go.uber.org/zap"
)

var _ common.ListenerLoader = &SlackLoader{}

func NewSlackLoader(messageTemplate, configString string, events []testkube.EventType) *SlackLoader {
	return &SlackLoader{
		Log:             log.DefaultLogger,
		messageTemplate: messageTemplate,
		events:          events,
		configString:    configString,
	}
}

// SlackLoader is a reconciler for websocket events for now it returns single listener for slack
type SlackLoader struct {
	Log             *zap.SugaredLogger
	messageTemplate string
	events          []testkube.EventType
	configString    string
}

func (r *SlackLoader) Kind() string {
	return "slack"
}

// Load returns single listener for slack (as we don't have any sophisticated config yet)
func (r *SlackLoader) Load() (listeners common.Listeners, err error) {
	var config []slack.NotificationsConfig
	if err := json.Unmarshal([]byte(r.configString), &config); err != nil {
		r.Log.Errorw("error unmarshalling slack config", "error", err)
	}
	slackNotifier := slack.NewNotifier(r.messageTemplate, config)
	if slackNotifier.Ready {
		return common.Listeners{NewSlackListener("slack", "", r.events, slackNotifier)}, nil
	}
	r.Log.Debugw("Slack notifier is not ready or not configured properly, omiting", "kind", r.Kind())
	return common.Listeners{}, nil
}
