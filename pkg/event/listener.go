package event

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

type ListenerKind string

func (k ListenerKind) String() string {
	return string(k)
}

const (
	ListenerKindWebsocket ListenerKind = "websocket"
	ListenerKindSlack     ListenerKind = "slack"
	ListenerKindWebhook   ListenerKind = "webhook"
)

type Listener interface {
	Notify(event testkube.TestkubeEvent) testkube.TestkubeEventResult
	Kind() ListenerKind
}

type ListenerReconiler interface {
	Load() []Listener
	Kind() ListenerKind
}
