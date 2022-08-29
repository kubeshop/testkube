package common

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

const (
	ListenerKindWebsocket string = "websocket"
	ListenerKindSlack     string = "slack"
	ListenerKindWebhook   string = "webhook"
)

type Listener interface {
	Notify(event testkube.TestkubeEvent) testkube.TestkubeEventResult
	Kind() string
	Selector() string
	Events() []testkube.TestkubeEventType
	Metadata() map[string]string
}

type ListenerReconiler interface {
	Load() (listeners []Listener, err error)
	Kind() string
}
