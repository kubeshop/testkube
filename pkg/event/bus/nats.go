package bus

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/nats-io/nats.go"
)

var _ Bus = &NATS{}

const (
	SubscribeBuffer = 100
)

func NewNATSEventBus(nc *nats.EncodedConn) *NATS {
	n := &NATS{
		nc: nc,
	}

	return n
}

type NATS struct {
	nc *nats.EncodedConn
}

func (n *NATS) Publish(event testkube.Event) error {
	return n.nc.Publish(event.Type_.String(), event)
}

func (n *NATS) Subscribe(eventType testkube.EventType, queueName string) (chan testkube.Event, error) {
	ch := make(chan testkube.Event, SubscribeBuffer)
	_, err := n.nc.QueueSubscribe(eventType.String(), queueName, func(event testkube.Event) {
		fmt.Printf("EVENT: %s Q: %s, E:%+v\n", eventType.String(), queueName, event)
		ch <- event
	})
	return ch, err
}
