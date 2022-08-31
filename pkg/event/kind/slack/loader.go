package slack

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

var _ common.ListenerLoader = &SlackLoader{}

func NewSlackLoader() *SlackLoader {
	return &SlackLoader{}
}

// SlackLoader is a reconciler for websocket events for now it returns single listener for slack
type SlackLoader struct {
}

func (r *SlackLoader) Kind() string {
	return "slack"
}

// Load returns single listener for slack (as we don't have any sophisticated config yet)
func (r *SlackLoader) Load() (listeners common.Listeners, err error) {
	// TODO handle slack notifications based on event types
	// for now implementation is just a single Slack Listener for all events
	return common.Listeners{
		NewSlackListener("", []testkube.TestkubeEventType{}),
	}, nil
}
