package slack

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
)

var _ common.ListenerLoader = &WebsocketLoader{}

func NewWebsocketLoader() *WebsocketLoader {
	return &WebsocketLoader{}
}

// WebsocketLoader is a reconciler for websocket events for now it returns single listener for slack
type WebsocketLoader struct {
}

func (r *WebsocketLoader) Kind() string {
	return "slack"
}

// Load returns single listener for slack (as we don't have any sophisticated config yet)
func (r *WebsocketLoader) Load() (listeners common.Listeners, err error) {
	return common.Listeners{NewSlackListener("", []testkube.TestkubeEventType{})}, nil
}
