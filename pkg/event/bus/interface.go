package bus

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

type Bus interface {
	Publish(event testkube.Event) error
	Subscribe(eventType testkube.EventType, queue string) (chan testkube.Event, error)
}
