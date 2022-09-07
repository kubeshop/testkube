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
	"github.com/nats-io/nats.go"
)

const (
	eventsBuffer      = 10000
	workersCount      = 20
	reconcileInterval = time.Second
)

// NewEmitter returns new emitter instance
func NewEmitter() *Emitter {

	// TODO move it to config
	nc, err := nats.Connect("localhost")
	if err != nil {
		log.DefaultLogger.Fatalw("error connecting to nats", "error", err)
	}

	// and automatic JSON encoder
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		log.DefaultLogger.Fatalw("error connecting to nats", "error", err)
	}

	return &Emitter{
		Events:  make(chan testkube.Event, eventsBuffer),
		Results: make(chan testkube.EventResult, eventsBuffer),
		Log:     log.DefaultLogger,
		Bus:     bus.NewNATSEventBus(ec),
	}
}

// Emitter handles events emitting for webhooks
type Emitter struct {
	Events    chan testkube.Event
	Results   chan testkube.EventResult
	Listeners common.Listeners
	Loader    Loader
	Log       *zap.SugaredLogger
	mutex     sync.Mutex
	Bus       bus.Bus
}

// Register adds new listener
func (e *Emitter) Register(listener common.Listener) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.Listeners = append(e.Listeners, listener)
}

// UpdateListeners updates listeners list
func (e *Emitter) UpdateListeners(listeners common.Listeners) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for i, new := range e.Listeners {
		found := false
		for j, old := range e.Listeners {
			if new.Name() == old.Name() {
				e.Listeners[i] = listeners[j]
				found = true
			}
		}
		if !found {
			e.Listeners = append(e.Listeners, listeners[i])
		}
	}

	e.Listeners = listeners
}

// Notify notifies emitter with webhook
func (e *Emitter) Notify(event testkube.Event) {
	err := e.Bus.Publish(event)
	if err != nil {
		e.Log.Errorw("error publishing event", event.Log()...)
	}
}

// Listen runs emitter workers responsible for sending HTTP requests
func (e *Emitter) Listen() {
	for _, l := range e.Listeners {
		go func(l common.Listener) {
			log := e.Log.With("listener", l.Event(), "name", l.Name(), "selector", l.Selector())
			log.Infow("starting listener")
			events, err := e.Bus.Subscribe(l.Event(), l.Name())
			if err != nil {
				log.Errorw("error subscribing to event", "event", l.Event(), "name", l.Name(), "error", err)
				return
			}

			for event := range events {
				log.Debugw("received event", event.Log()...)
				if event.Valid(l.Selector()) {
					log.Infow("event matching", event.Log()...)
					// TODO consider scaling to go routines
					e.Results <- l.Notify(event)
				} else {
					log.Debugw("event not matching selector", event.Log()...)
				}
			}
		}(l)
	}
}

// Reconcile reloads listeners from all registered reconcilers
func (s *Emitter) Reconcile(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.Log.Infow("stopping reconciler")
			return
		default:
			listeners := s.Loader.Reconcile()
			s.UpdateListeners(listeners)
			s.Log.Debugw("reconciled listeners", s.Listeners.Log()...)
			time.Sleep(reconcileInterval)
		}
	}
}
