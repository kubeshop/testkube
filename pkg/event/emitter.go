package event

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
)

const (
	reconcileInterval = time.Second
)

// NewEmitter returns new emitter instance
func NewEmitter(eventBus bus.Bus, clusterName string) *Emitter {
	return &Emitter{
		loader:      NewLoader(),
		log:         log.DefaultLogger,
		bus:         eventBus,
		listeners:   make(common.Listeners, 0),
		clusterName: clusterName,
	}
}

type Interface interface {
	Notify(event testkube.Event)
}

// Emitter handles events emitting for webhooks
type Emitter struct {
	*loader

	log         *zap.SugaredLogger
	listeners   common.Listeners
	mutex       sync.RWMutex
	bus         bus.Bus
	clusterName string
}

// Register adds new listener
func (e *Emitter) Register(listener common.Listener) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.listeners = append(e.listeners, listener)
}

// updateListeners updates listeners list
func (e *Emitter) updateListeners(listeners common.Listeners) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	result := make([]common.Listener, 0)

	oldMap := listerersToMap(e.listeners)
	newMap := listerersToMap(listeners)

	// check for missing listeners
	for kind, lMap := range oldMap {
		// clean missing kinds
		if _, ok := newMap[kind]; !ok {
			for _, l := range lMap {
				e.stopListener(l.Name())
			}

			continue
		}

		// stop missing listeners
		for name, l := range lMap {
			if _, ok := newMap[kind][name]; !ok {
				e.stopListener(l.Name())
			}
		}
	}

	// check for new listeners
	for kind, lMap := range newMap {
		// start all listeners for new kind
		if _, ok := oldMap[kind]; !ok {
			for _, l := range lMap {
				e.startListener(l)
				result = append(result, l)
			}

			continue
		}

		// start new listeners and restart updated ones
		for name, l := range lMap {
			if current, ok := oldMap[kind][name]; !ok {
				e.startListener(l)
			} else {
				if !common.CompareListeners(current, l) {
					e.stopListener(current.Name())
					e.startListener(l)
				}
			}

			result = append(result, l)
		}
	}

	e.listeners = result
}

func listerersToMap(listeners []common.Listener) map[string]map[string]common.Listener {
	m := make(map[string]map[string]common.Listener, 0)

	for _, l := range listeners {
		if _, ok := m[l.Kind()]; !ok {
			m[l.Kind()] = make(map[string]common.Listener, 0)
		}

		m[l.Kind()][l.Name()] = l
	}

	return m
}

// Notify notifies emitter with webhook
func (e *Emitter) Notify(event testkube.Event) {
	// TODO(emil): what does specifying cluster name do here? is this used anywhere? does this have signficance to nats?
	event.ClusterName = e.clusterName
	// TODO(emil): log a warning if the topic is not matching the subscribe topic for the emitter
	err := e.bus.PublishTopic(event.Topic(), event)
	if err != nil {
		e.log.Errorw("error publishing event", append(event.Log(), "error", err))
		return
	}
	e.log.Debugw("event published", event.Log()...)
}

// Listen runs emitter workers responsible for sending HTTP requests
func (e *Emitter) Listen(ctx context.Context) {
	// clean after closing Emitter
	go func() {
		<-ctx.Done()
		e.log.Warn("closing event bus")

		for _, l := range e.listeners {
			go e.bus.Unsubscribe(l.Name())
		}

		e.bus.Close()
	}()

	e.mutex.Lock()
	defer e.mutex.Unlock()

	for _, l := range e.listeners {
		go e.startListener(l)
	}
}

func (e *Emitter) startListener(l common.Listener) {
	// TODO(emil): why are we creating a subscription to the same topic for all these listeners, and then coding all this logic to start and stop listeners
	// NOTE(emil): the topic where the listeners events come in on
	err := e.bus.SubscribeTopic("agentevents.>", l.Name(), e.notifyHandler(l))
	if err != nil {
		e.log.Errorw("error while starting listener", "error", err)
	}
	e.log.Infow("started listener", l.Name(), l.Metadata())
}

func (e *Emitter) stopListener(name string) {
	err := e.bus.Unsubscribe(name)
	if err != nil {
		e.log.Errorw("error while stopping listener", "error", err)
	}
	e.log.Info("stopped listener", name)
}

func (e *Emitter) notifyHandler(l common.Listener) bus.Handler {
	logger := e.log.With("listen-on", l.Events(), "queue-group", l.Name(), "selector", l.Selector(), "metadata", l.Metadata())
	return func(event testkube.Event) error {
		if !l.Match(event) {
			log.Tracew(logger, "dropping event not matching selector or type", event.Log()...)
			return nil
		}
		// Event type fanout
		// NOTE(emil): This fanout behavior is old, but kept in tact because it
		// is not of priority - can an event even match multiple event types?
		// and even if it does should it fire multiple events for the same
		// listener? even then does this fanout logic not belong in the
		// listener notify implementation where each one might decide to handle
		// this differently?
		matchedEventTypes, _ := event.Valid(l.Selector(), l.Events())
		for i := range matchedEventTypes {
			event.Type_ = &matchedEventTypes[i]
			// TODO(emil): note these results are just logged, not sure there is much point even returning them, can just log in the listener
			result := l.Notify(event)
			log.Tracew(logger, "listener notified", append(event.Log(), "result", result)...)
		}
		return nil
	}
}

// Reconcile reloads listeners from all registered reconcilers
func (e *Emitter) Reconcile(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			e.log.Info("stopping reconciler")
			return
		default:
			listeners := e.loader.Reconcile()
			e.updateListeners(listeners)
			log.Tracew(e.log, "reconciled listeners", e.ListenersDump()...)
			time.Sleep(reconcileInterval)
		}
	}
}

// ListenersDump dumps all the currently reconciled listeners in an array for debugging.
func (e *Emitter) ListenersDump() []any {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.listeners.Dump()
}
