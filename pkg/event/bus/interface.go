package bus

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

type Handler func(event testkube.Event) error

type Bus interface {
	Publish(event testkube.Event) error
	Subscribe(eventType testkube.EventType, queue string, handler Handler) error
	Unsubscribe(eventType testkube.EventType, queue string) error
	Close() error
}
