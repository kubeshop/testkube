//nolint:staticcheck
package bus

import (
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/nats-io/nats.go"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/utils"
)

var (
	_ Bus = (*NATSBus)(nil)

	NATS_RETRY_ATTEMPTS uint = 20
)

const (
	SubscribeBuffer        = 1
	SubscriptionName       = "agentevents"
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
	opts = []nats.Option{
		// Never stop trying to reconnect — the process should not require a restart
		// due to a transient NATS outage.
		// Note: RetryOnFailedConnect is intentionally omitted. It would make
		// nats.Connect() return nil on an unreachable server, silently disabling
		// the retry.DoWithData startup loop and removing crash-on-startup behaviour.
		// MaxReconnects(-1) covers all runtime reconnection; startup retries are
		// handled by the caller's retry loop.
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2 * time.Second),
	}

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
	nc, err := NewNATSConnection(cfg, opts...)
	if err != nil {
		return nil, err
	}

	// automatic NATS JSON CODEC
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}

	return ec, nil
}

func NewNATSConnection(cfg ConnectionConfig, opts ...nats.Option) (*nats.Conn, error) {
	opts = append(opts, optsFromConfig(cfg)...)
	opts = append(opts,
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.DefaultLogger.Warnw("nats disconnected",
				"error_type", "nats_connection_closed",
				"error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.DefaultLogger.Infow("nats reconnected",
				"url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			log.DefaultLogger.Errorw("nats connection permanently closed",
				"error_type", "nats_connection_closed")
		}),
	)

	nc, err := retry.DoWithData(
		func() (*nats.Conn, error) {
			return nats.Connect(cfg.NatsURI, opts...)
		},
		retry.DelayType(retry.FixedDelay),
		retry.Delay(utils.DefaultRetryDelay),
		retry.Attempts(NATS_RETRY_ATTEMPTS),
	)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

func NewNATSBus(nc *nats.EncodedConn, cfg ConnectionConfig) *NATSBus {
	return &NATSBus{
		nc:  nc,
		cfg: cfg,
	}
}

// subscriptionEntry holds everything needed to re-register a subscription on a
// fresh connection after a reconnect.
// When queue is empty a plain Subscribe is used; otherwise QueueSubscribe.
type subscriptionEntry struct {
	topic   string
	queue   string // empty → plain Subscribe, non-empty → QueueSubscribe
	handler Handler
	sub     *nats.Subscription
}

type NATSBus struct {
	nc            *nats.EncodedConn
	cfg           ConnectionConfig
	mu            sync.RWMutex
	subscriptions sync.Map // map[string]*subscriptionEntry
}

// Publish publishes event to NATS on events topic
func (n *NATSBus) Publish(event testkube.Event) error {
	return n.PublishTopic(SubscriptionName, event)
}

// Subscribe subscribes to NATS events topic
func (n *NATSBus) Subscribe(queueName string, handler Handler) error {
	return n.SubscribeTopic(SubscriptionName, queueName, handler)
}

// reconnect replaces the underlying connection when it has been permanently
// closed.  Callers must NOT hold n.mu when calling this.
func (n *NATSBus) reconnect() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Another goroutine may have already reconnected while we waited for the lock.
	if !n.nc.Conn.IsClosed() {
		return nil
	}

	if n.cfg.NatsURI == "" {
		return errors.New("nats reconnect: no URI configured (embedded connection cannot reconnect)")
	}

	log.DefaultLogger.Warnw("nats connection is closed, reinitialising",
		"error_type", "nats_connection_closed",
		"url", n.cfg.NatsURI)

	conn, err := NewNATSEncodedConnection(n.cfg)
	if err != nil {
		return fmt.Errorf("nats reconnect failed: %w", err)
	}

	// Re-register subscriptions BEFORE exposing conn via n.nc.  This closes the
	// window where the new connection is live but has no handlers — messages
	// published to subscribed topics during that gap would otherwise be silently
	// discarded by the NATS server.
	n.subscriptions.Range(func(key, value any) bool {
		entry := value.(*subscriptionEntry)

		var newSub *nats.Subscription
		var serr error
		if entry.queue == "" {
			newSub, serr = conn.Subscribe(entry.topic, entry.handler)
		} else {
			newSub, serr = conn.QueueSubscribe(entry.topic, entry.queue, entry.handler)
		}
		if serr != nil {
			log.DefaultLogger.Errorw("failed to re-register subscription after nats reconnect",
				"topic", entry.topic,
				"queue", entry.queue,
				"error", serr)
			return true
		}
		n.subscriptions.Store(key, &subscriptionEntry{
			topic:   entry.topic,
			queue:   entry.queue,
			handler: entry.handler,
			sub:     newSub,
		})
		return true
	})

	n.nc = conn
	return nil
}

// PublishTopic publishes event to NATS on given topic.
// If the connection is permanently closed it attempts a single reconnect before
// giving up, so a transient NATS restart does not require a pod restart.
func (n *NATSBus) PublishTopic(topic string, event testkube.Event) error {
	log.Tracew(log.DefaultLogger, "publishing event", event.Log()...)

	n.mu.RLock()
	nc := n.nc
	n.mu.RUnlock()

	err := nc.Publish(topic, event)
	if err == nil {
		return nil
	}

	if !errors.Is(err, nats.ErrConnectionClosed) {
		return err
	}

	// ErrConnectionClosed means the connection is permanently gone — not a
	// transient blip.  With MaxReconnects(-1) the NATS library handles transient
	// outages internally (buffering publishes while reconnecting), so this path
	// is a last-resort safety net for the rare case where the connection object
	// itself must be replaced (e.g. after an explicit Close or an unforeseen
	// client state machine edge-case).
	log.DefaultLogger.Warnw("nats connection closed during publish, attempting reconnect",
		"topic", topic,
		"error_type", "nats_connection_closed")

	if rerr := n.reconnect(); rerr != nil {
		return fmt.Errorf("nats publish failed, reconnect error: %w", rerr)
	}

	n.mu.RLock()
	nc = n.nc
	n.mu.RUnlock()

	return nc.Publish(topic, event)
}

// SubscribeTopic subscribes to NATS topic
func (n *NATSBus) SubscribeTopic(topic, queueName string, handler Handler) error {
	// sanitize names for NATS
	queue := common.ListenerName(queueName)

	n.mu.RLock()
	nc := n.nc
	n.mu.RUnlock()

	// async subscribe on queue
	s, err := nc.QueueSubscribe(topic, queue, handler)
	if err == nil {
		// store topic, queue, and handler so reconnect() can re-register them
		key := n.queueName(SubscriptionName, queue)
		n.subscriptions.Store(key, &subscriptionEntry{
			topic:   topic,
			queue:   queue,
			handler: handler,
			sub:     s,
		})
	}

	return err
}

func (n *NATSBus) Unsubscribe(queueName string) error {
	// sanitize names for NATS
	queue := common.ListenerName(queueName)

	key := n.queueName(SubscriptionName, queue)
	if v, ok := n.subscriptions.LoadAndDelete(key); ok {
		return v.(*subscriptionEntry).sub.Drain()
	}
	return nil
}

func (n *NATSBus) Close() error {
	n.mu.RLock()
	nc := n.nc
	n.mu.RUnlock()
	nc.Close()
	return nil
}

func (n *NATSBus) queueName(subscription, queue string) string {
	return fmt.Sprintf("%s.%s", subscription, queue)
}

func (n *NATSBus) TraceEvents() {
	n.mu.RLock()
	nc := n.nc
	n.mu.RUnlock()

	topic := SubscriptionName + ".>"
	handler := Handler(func(event testkube.Event) error {
		log.Tracew(log.DefaultLogger, "all events.> trace", event.Log()...)
		return nil
	})

	s, err := nc.Subscribe(topic, handler)
	if err != nil {
		log.DefaultLogger.Errorw("error subscribing to all events", "error", err)
		return
	}

	// Store with empty queue so reconnect() re-registers it via plain Subscribe.
	n.subscriptions.Store("trace:"+topic, &subscriptionEntry{
		topic:   topic,
		queue:   "",
		handler: handler,
		sub:     s,
	})

	log.DefaultLogger.Infow("subscribed to all events", "subscription", s.Subject)
}
