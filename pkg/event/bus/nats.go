package bus

import (
	"fmt"
	"sync"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/nats-io/nats.go"
)

var _ Bus = &NATS{}

const (
	SubscribeBuffer  = 1
	SubscriptionName = "events"
)

func NewNATSEventBus(nc *nats.EncodedConn) *NATS {
	n := &NATS{
		nc: nc,
	}

	return n
}

type NATS struct {
	nc            *nats.EncodedConn
	subscriptions sync.Map
}

func (n *NATS) Publish(event testkube.Event) error {
	return n.nc.Publish(SubscriptionName, event)
}

func (n *NATS) Subscribe(queueName string, handler Handler) error {
	// sanitize names for NATS
	queue := common.ListenerName(queueName)

	// async subscribe on queue
	s, err := n.nc.QueueSubscribe(SubscriptionName, queue, handler)

	// store subscription for later unsubscribe
	key := fmt.Sprintf("%s.%s", SubscriptionName, queue)
	n.subscriptions.Store(key, s)

	return err
}

func (n *NATS) Unsubscribe(queueName string) error {
	// sanitize names for NATS
	queue := common.ListenerName(queueName)

	key := fmt.Sprintf("%s.%s", SubscriptionName, queue)
	if s, ok := n.subscriptions.Load(key); ok {
		s.(*nats.Subscription).Unsubscribe()
	}
	return nil
}

func (n *NATS) Close() error {
	n.nc.Close()
	return nil
}
