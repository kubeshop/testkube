package common

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	ListenerKindWebsocket string = "websocket"
	ListenerKindSlack     string = "slack"
	ListenerKindWebhook   string = "webhook"
)

type Listener interface {
	// Name uniquely identifies listener
	Name() string
	// Notify sends event to listener
	Notify(event testkube.Event) testkube.EventResult
	// Kind of listener
	Kind() string
	// Selector is used to filter events
	Selector() string
	// Event is used to filter events
	Events() []testkube.EventType
	// Metadata with additional information about listener
	Metadata() map[string]string
}

type ListenerLoader interface {
	// Load listeners from configuration
	Load() (listeners Listeners, err error)
	// Kind of listener
}

type Listeners []Listener

func (l Listeners) Log() []any {
	var result []any
	for _, listener := range l {
		result = append(result, map[string]any{
			"kind":     listener.Kind(),
			"events":   listener.Events(),
			"selector": listener.Selector(),
			"metadata": listener.Metadata(),
		})
	}
	return []any{"listeners", result}
}

// CompareListeners compares listeners by metadata
func CompareListeners(a, b Listener) bool {
	mapA := a.Metadata()
	mapB := b.Metadata()

	for key, value := range mapA {
		if v, ok := mapB[key]; !ok || value != v {
			return false
		}
	}

	for key, value := range mapB {
		if v, ok := mapA[key]; !ok || value != v {
			return false
		}
	}

	return true
}
