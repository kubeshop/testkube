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
	eventsBuffer      = 10000
	workersCount      = 20
	reconcileInterval = time.Second
)

// NewEmitter returns new emitter instance
func NewEmitter(eventBus bus.Bus) *Emitter {
	return &Emitter{
		Results:   make(chan testkube.EventResult, eventsBuffer),
		Log:       log.DefaultLogger,
		Loader:    NewLoader(),
		Bus:       eventBus,
		Listeners: make(common.Listeners, 0),
	}
}

// Emitter handles events emitting for webhooks
type Emitter struct {
	Results   chan testkube.EventResult
	Listeners common.Listeners
	Loader    *Loader
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

	for i, new := range listeners {
		found := false
		for j, old := range e.Listeners {
			if new.Name() == old.Name() {
				e.Listeners[j] = listeners[i]
				found = true
			}
		}
		// if listener is not registered yet we need to subscribe
		if !found {
			e.Listeners = append(e.Listeners, listeners[i])
			e.startListener(listeners[i])
		}
	}
}

// Notify notifies emitter with webhook
func (e *Emitter) Notify(event testkube.Event) {
	err := e.Bus.Publish(event)
	e.Log.Infow("event published", append(event.Log(), "error", err)...)
}

// Listen runs emitter workers responsible for sending HTTP requests
func (e *Emitter) Listen(ctx context.Context) {
	// clean after closing Emitter
	go func() {
		<-ctx.Done()
		e.Log.Warn("closing event bus")

		for _, l := range e.Listeners {
			go e.Bus.Unsubscribe(l.Name())
		}

		e.Bus.Close()
	}()

	e.mutex.Lock()
	defer e.mutex.Unlock()

	for _, l := range e.Listeners {
		go e.startListener(l)
	}
}

func (e *Emitter) startListener(l common.Listener) {
	e.Log.Infow("starting listener", l.Name(), l.Metadata())
	err := e.Bus.Subscribe(l.Name(), e.notifyHandler(l))
	if err != nil {
		e.Log.Errorw("error subscribing to event", "error", err)
	}
}

func (e *Emitter) notifyHandler(l common.Listener) bus.Handler {
	log := e.Log.With("listen-on", l.Events(), "queue-group", l.Name(), "selector", l.Selector(), "metadata", l.Metadata())
	return func(event testkube.Event) error {
		if event.Valid(l.Selector(), l.Events()) {
			l.Notify(event)
			log.Infow("listener notified", "event", event.Id, "executionId", event.TestExecution.Id)
		} else {
			log.Infow("dropping event not matching selector or type", event.Log()...)
		}
		return nil
	}
}

// Reconcile reloads listeners from all registered reconcilers
func (e *Emitter) Reconcile(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			e.Log.Infow("stopping reconciler")
			return
		default:
			listeners := e.Loader.Reconcile()
			e.UpdateListeners(listeners)
			e.Log.Debugw("reconciled listeners", e.Listeners.Log()...)
			time.Sleep(reconcileInterval)

		}
	}
}
