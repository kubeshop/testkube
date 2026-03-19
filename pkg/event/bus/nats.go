//nolint:staticcheck
package bus

import (
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/avast/retry-go/v5"
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

	natsMaxReconnects = -1
	natsReconnectWait = 2 * time.Second
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
		nats.MaxReconnects(natsMaxReconnects),
		nats.ReconnectWait(natsReconnectWait),
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

	nc, err := retry.NewWithData[*nats.Conn](
		retry.DelayType(retry.FixedDelay),
		retry.Delay(utils.DefaultRetryDelay),
		retry.Attempts(NATS_RETRY_ATTEMPTS),
	).Do(func() (*nats.Conn, error) {
		return nats.Connect(cfg.NatsURI, opts...)
	})
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
	connMu        sync.RWMutex
	reconnectMu   sync.Mutex // serialises reconnect attempts; never held while connMu is held
	subscriptions sync.Map   // map[string]*subscriptionEntry
}

// getNC returns the current encoded connection under a read lock.
func (n *NATSBus) getNC() *nats.EncodedConn {
	n.connMu.RLock()
	defer n.connMu.RUnlock()
	return n.nc
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
// closed.  Callers must NOT hold n.connMu when calling this.
//
// Design notes:
//   - reconnectMu serialises concurrent reconnect attempts so only one goroutine
//     pays the cost of NewNATSEncodedConnection's retry loop at a time.
//   - connMu (the read/write lock guarding n.nc) is held only for the final
//     pointer swap, so publishers are not stalled for the full retry duration.
//   - CompareAndSwap is used when updating subscription entries so that a
//     concurrent Unsubscribe (which calls LoadAndDelete) wins: if the key was
//     deleted between Range seeing it and the Store, the orphan subscription is
//     drained rather than re-inserted.
func (n *NATSBus) reconnect() error {
	if n.cfg.NatsURI == "" {
		return errors.New("nats reconnect: no URI configured (embedded connection cannot reconnect)")
	}

	// Serialise reconnect attempts.  If two goroutines both detect
	// ErrConnectionClosed, only one should create a new connection.
	n.reconnectMu.Lock()
	defer n.reconnectMu.Unlock()

	// Re-check now that we hold reconnectMu: a previous waiter may have
	// already swapped in a healthy connection.
	if !n.getNC().Conn.IsClosed() {
		return nil
	}

	log.DefaultLogger.Warnw("nats connection is closed, reinitialising",
		"error_type", "nats_connection_closed",
		"url", n.cfg.NatsURI)

	// Create the new connection outside connMu so that publishers are not
	// stalled during the (potentially slow) retry loop inside
	// NewNATSEncodedConnection.  (reconnectMu is still held to serialise
	// concurrent reconnect attempts.)
	conn, err := NewNATSEncodedConnection(n.cfg)
	if err != nil {
		return fmt.Errorf("nats reconnect failed: %w", err)
	}

	// Re-register subscriptions on the new connection BEFORE exposing it via
	// n.nc.  This closes the window where messages could arrive on a topic that
	// has no handler yet.
	var failedKeys []any
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
			// Mark for removal: leaving a stale entry would cause silent message
			// loss and unexpected Drain() calls on dead subscriptions.
			failedKeys = append(failedKeys, key)
			return true
		}

		newEntry := &subscriptionEntry{
			topic:   entry.topic,
			queue:   entry.queue,
			handler: entry.handler,
			sub:     newSub,
		}
		// CompareAndSwap against the exact pointer seen by Range.  If
		// Unsubscribe ran LoadAndDelete between Range seeing this entry and now,
		// the swap will fail and we drain the orphan subscription rather than
		// silently re-inserting a ghost entry that can never be removed.
		if !n.subscriptions.CompareAndSwap(key, value, newEntry) {
			_ = newSub.Drain()
		}
		return true
	})

	for _, key := range failedKeys {
		n.subscriptions.Delete(key)
	}

	// Hold the write lock only for the pointer swap.
	n.connMu.Lock()
	n.nc = conn
	n.connMu.Unlock()
	return nil
}

// PublishTopic publishes event to NATS on given topic.
// If the connection is permanently closed it attempts a single reconnect before
// giving up, so a transient NATS restart does not require a pod restart.
func (n *NATSBus) PublishTopic(topic string, event testkube.Event) error {
	log.Tracew(log.DefaultLogger, "publishing event", event.Log()...)

	nc := n.getNC()
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

	return n.getNC().Publish(topic, event)
}

// SubscribeTopic subscribes to NATS topic.
// If the connection is permanently closed it attempts a single reconnect before
// giving up, mirroring the self-healing behaviour of PublishTopic.
func (n *NATSBus) SubscribeTopic(topic, queueName string, handler Handler) error {
	// sanitize names for NATS
	queue := common.ListenerName(queueName)

	// async subscribe on queue
	s, err := n.getNC().QueueSubscribe(topic, queue, handler)
	if err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
		return err
	}
	if err != nil {
		log.DefaultLogger.Warnw("nats connection closed during subscribe, attempting reconnect",
			"topic", topic,
			"error_type", "nats_connection_closed")
		if rerr := n.reconnect(); rerr != nil {
			return fmt.Errorf("nats subscribe failed, reconnect error: %w", rerr)
		}
		s, err = n.getNC().QueueSubscribe(topic, queue, handler)
		if err != nil {
			return err
		}
	}

	// store topic, queue, and handler so reconnect() can re-register them
	key := n.queueName(SubscriptionName, queue)
	n.subscriptions.Store(key, &subscriptionEntry{
		topic:   topic,
		queue:   queue,
		handler: handler,
		sub:     s,
	})
	return nil
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
	n.getNC().Close()
	return nil
}

func (n *NATSBus) queueName(subscription, queue string) string {
	return fmt.Sprintf("%s.%s", subscription, queue)
}

func (n *NATSBus) TraceEvents() {
	topic := SubscriptionName + ".>"
	handler := Handler(func(event testkube.Event) error {
		log.Tracew(log.DefaultLogger, "all events.> trace", event.Log()...)
		return nil
	})

	s, err := n.getNC().Subscribe(topic, handler)
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
