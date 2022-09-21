package bus

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

type Handler func(event testkube.Event) error

type Bus interface {
	Publish(event testkube.Event) error
	Subscribe(queue string, handler Handler) error
	Unsubscribe(queue string) error
	Close() error
}
