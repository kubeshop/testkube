package bus

import (
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
)

var (
	_ Bus = (*NATSBus)(nil)
)

const (
	SubscribeBuffer        = 1
	SubscriptionName       = "events"
	InternalPublishTopic   = "internal.all"
	InternalSubscribeTopic = "internal.>"
)

type ConnectionConfig struct {
	NatsURI            string
	NatsSecure         bool
	NatsSkipVerify     bool
	NatsCertFile       string
	NatsKeyFile        string
	NatsCAFile         string
	NatsConnectTimeout time.Duration
}

func optsFromConfig(cfg ConnectionConfig) (opts []nats.Option) {
	opts = []nats.Option{}
	if cfg.NatsSecure {
		if cfg.NatsSkipVerify {
			opts = append(opts, nats.Secure(&tls.Config{InsecureSkipVerify: true}))
		} else {
			opts = append(opts, nats.ClientCert(cfg.NatsCertFile, cfg.NatsKeyFile))
			if cfg.NatsCAFile != "" {
				opts = append(opts, nats.RootCAs(cfg.NatsCAFile))
			}
		}
	}

	if cfg.NatsConnectTimeout > 0 {
		opts = append(opts, nats.Timeout(cfg.NatsConnectTimeout))
	}

	return opts
}

func NewNATSEncodedConnection(cfg ConnectionConfig, opts ...nats.Option) (*nats.EncodedConn, error) {
	opts = append(opts, optsFromConfig(cfg)...)

	nc, err := NewNATSConnection(cfg, opts...)
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

	if err != nil {
		log.DefaultLogger.Errorw("error creating NATS connection", "error", err)
	}

	return ec, nil
}

func NewNATSConnection(cfg ConnectionConfig, opts ...nats.Option) (*nats.Conn, error) {
	opts = append(opts, optsFromConfig(cfg)...)

	nc, err := nats.Connect(cfg.NatsURI, opts...)
	if err != nil {
		log.DefaultLogger.Fatalw("error connecting to nats", "error", err)
		return nil, err
	}

	return nc, nil
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

// Publish publishes event to NATS on events topic
func (n *NATSBus) Publish(event testkube.Event) error {
	return n.PublishTopic(SubscriptionName, event)
}

// Subscribe subscribes to NATS events topic
func (n *NATSBus) Subscribe(queueName string, handler Handler) error {
	return n.SubscribeTopic(SubscriptionName, queueName, handler)
}

// PublishTopic publishes event to NATS on given topic
func (n *NATSBus) PublishTopic(topic string, event testkube.Event) error {
	return n.nc.Publish(topic, event)
}

// SubscribeTopic subscribes to NATS topic
func (n *NATSBus) SubscribeTopic(topic, queueName string, handler Handler) error {
	// sanitize names for NATS
	queue := common.ListenerName(queueName)

	// async subscribe on queue
	s, err := n.nc.QueueSubscribe(topic, queue, handler)

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
