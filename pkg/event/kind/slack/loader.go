package slack

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/slacknotifier"
	"go.uber.org/zap"
)

var _ common.ListenerLoader = &SlackLoader{}

func NewSlackLoader() *SlackLoader {
	return &SlackLoader{
		Log: log.DefaultLogger,
	}
}

// SlackLoader is a reconciler for websocket events for now it returns single listener for slack
type SlackLoader struct {
	Log *zap.SugaredLogger
}

func (r *SlackLoader) Kind() string {
	return "slack"
}

// Load returns single listener for slack (as we don't have any sophisticated config yet)
func (r *SlackLoader) Load() (listeners common.Listeners, err error) {
	// TODO handle slack notifications based on event types, for now we just load all
	if slacknotifier.Ready {
		return common.Listeners{NewSlackListener("slack", "", testkube.AllEventTypes)}, nil
	}
	r.Log.Debugw("Slack notifier is not ready or not configured properly, omiting", "kind", r.Kind())
	return common.Listeners{}, nil
}
