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

type ListenerLoader interface {
	Load() (listeners Listeners, err error)
	Kind() string
}

type Listeners []Listener

func (l Listeners) Log() []any {
	var result []any
	for _, listener := range l {
		result = append(result, map[string]any{
			"kind":     listener.Kind(),
			"selector": listener.Selector(),
			"metadata": listener.Metadata(),
		})
	}
	return result
}
