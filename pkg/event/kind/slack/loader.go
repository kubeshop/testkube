package slack

import (
	"encoding/json"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/slack"
)

var _ common.ListenerLoader = (*SlackLoader)(nil)

func NewSlackLoader(messageTemplate, configString, clusterName, dashboardURI string,
	events []testkube.EventType, envs map[string]string) *SlackLoader {
	var config []slack.NotificationsConfig
	if err := json.Unmarshal([]byte(configString), &config); err != nil {
		log.DefaultLogger.Errorw("error unmarshalling slack config", "error", err)
	}
	slackNotifier := slack.NewNotifier(messageTemplate, clusterName, dashboardURI, config, envs)
	return &SlackLoader{
		Log:           log.DefaultLogger,
		events:        events,
		slackNotifier: slackNotifier,
	}
}

// SlackLoader is a reconciler for slack events for now it returns single listener for slack
type SlackLoader struct {
	Log           *zap.SugaredLogger
	events        []testkube.EventType
	slackNotifier *slack.Notifier
}

func (r *SlackLoader) Kind() string {
	return "slack"
}

// Load returns single listener for slack (as we don't have any sophisticated config yet)
func (r *SlackLoader) Load() (listeners common.Listeners, err error) {

	if r.slackNotifier.Ready {
		return common.Listeners{NewSlackListener("slack", "", r.events, r.slackNotifier)}, nil
	}
	r.Log.Debugw("Slack notifier is not ready or not configured properly, omiting", "kind", r.Kind())
	return common.Listeners{}, nil
}
