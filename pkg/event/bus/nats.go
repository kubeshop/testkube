package bus

import (
	"fmt"
	"os"
	"sync"

	"github.com/nats-io/nats.go"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
)

var _ Bus = &NATSBus{}

const (
	SubscribeBuffer  = 1
	SubscriptionName = "events"
)

func NewNATSConnection() (*nats.EncodedConn, error) {
	natsURI := "localhost"
	if uri, ok := os.LookupEnv("NATS_URI"); ok {
		natsURI = uri
	}

	nc, err := nats.Connect(natsURI)
	if err != nil {
		log.DefaultLogger.Fatalw("error connecting to nats", "error", err)
		return nil, err
	}

	// automatic NATS JSON CODEC
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		log.DefaultLogger.Fatalw("error connecting to nats", "error", err)
		return nil, err
	}

	return ec, nil
}

func NewNATSBus(nc *nats.EncodedConn) *NATSBus {
	return &NATSBus{
		nc: nc,
	}
}

type NATSBus struct {
	nc            *nats.EncodedConn
	subscriptions sync.Map
}

func (n *NATSBus) Publish(event testkube.Event) error {
	return n.nc.Publish(SubscriptionName, event)
}

func (n *NATSBus) Subscribe(queueName string, handler Handler) error {
	// sanitize names for NATS
	queue := common.ListenerName(queueName)

	// async subscribe on queue
	s, err := n.nc.QueueSubscribe(SubscriptionName, queue, handler)

	if err == nil {
		// store subscription for later unsubscribe
		key := n.queueName(SubscriptionName, queue)
		n.subscriptions.Store(key, s)
	}

	return err
}

func (n *NATSBus) Unsubscribe(queueName string) error {
	// sanitize names for NATS
	queue := common.ListenerName(queueName)

	key := n.queueName(SubscriptionName, queue)
	if s, ok := n.subscriptions.Load(key); ok {
		return s.(*nats.Subscription).Drain()
	}
	return nil
}

func (n *NATSBus) Close() error {
	n.nc.Close()
	return nil
}

func (n *NATSBus) queueName(subscription, queue string) string {
	return fmt.Sprintf("%s.%s", SubscriptionName, queue)
}
