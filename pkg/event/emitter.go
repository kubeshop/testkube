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
	reconcileInterval            = time.Second
	eventEmitterQueueName string = "emitter"
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

		err := e.bus.Unsubscribe(eventEmitterQueueName)
		if err != nil {
			e.log.Warnw("error while unsubscribing from emitted events", "error", err)
		}

		err = e.bus.Close()
		if err != nil {
			e.log.Warnw("error while closing event bus", "error", err)
		}
		e.log.Debug("closed event bus")
	}()

	err := e.bus.SubscribeTopic("agentevents.>", eventEmitterQueueName, e.eventHandler)
	if err != nil {
		e.log.Errorw("error while starting to listen for events", "error", err)
	}
	e.log.Infow("started listening for events")
}

func (e *Emitter) eventHandler(event testkube.Event) error {
	// Current set of listeners
	e.mutex.Lock()
	listeners := make([]common.Listener, len(e.listeners))
	copy(listeners, e.listeners)
	e.mutex.Unlock()
	// Find listeners that match the event
	for _, l := range e.listeners {
		// TODO(emil): this logging should be moved to listener
		logger := e.log.With("listen-on", l.Events(), "selector", l.Selector(), "metadata", l.Metadata())
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
			go func(notifyEvent testkube.Event, notifyLogger *zap.SugaredLogger) {
				// TODO(emil): note these results are just logged, not sure there is much point even returning them, can just log in the listener and all this can be simplified also
				result := l.Notify(notifyEvent)
				log.Tracew(notifyLogger, "listener notified", append(notifyEvent.Log(), "result", result)...)
			}(event, logger)
		}
	}
	return nil
}

// Reconcile reloads listeners from all registered reconcilers
func (e *Emitter) Reconcile(ctx context.Context) {
	ticker := time.NewTicker(reconcileInterval)
	for {
		select {
		case <-ctx.Done():
			e.log.Info("stopping reconciler")
			return
		case <-ticker.C:
			listeners := e.loader.Reconcile()
			e.mutex.Lock()
			e.listeners = listeners
			e.mutex.Unlock()
			log.Tracew(e.log, "reconciled listeners", e.ListenersDump()...)
		}
	}
}

// ListenersDump dumps all the currently reconciled listeners in an array for debugging.
func (e *Emitter) ListenersDump() []any {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.listeners.Dump()
}
