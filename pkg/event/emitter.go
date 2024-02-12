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
func NewEmitter(eventBus bus.Bus, clusterName string, envs map[string]string) *Emitter {
	return &Emitter{
		Results:     make(chan testkube.EventResult, eventsBuffer),
		Log:         log.DefaultLogger,
		Loader:      NewLoader(),
		Bus:         eventBus,
		Listeners:   make(common.Listeners, 0),
		ClusterName: clusterName,
		Envs:        envs,
	}
}

// Emitter handles events emitting for webhooks
type Emitter struct {
	Results     chan testkube.EventResult
	Listeners   common.Listeners
	Loader      *Loader
	Log         *zap.SugaredLogger
	mutex       sync.RWMutex
	Bus         bus.Bus
	ClusterName string
	Envs        map[string]string
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

	oldMap := make(map[string]map[string]common.Listener, 0)
	newMap := make(map[string]map[string]common.Listener, 0)
	result := make([]common.Listener, 0)

	for _, l := range e.Listeners {
		if _, ok := oldMap[l.Kind()]; !ok {
			oldMap[l.Kind()] = make(map[string]common.Listener, 0)
		}

		oldMap[l.Kind()][l.Name()] = l
	}

	for _, l := range listeners {
		if _, ok := newMap[l.Kind()]; !ok {
			newMap[l.Kind()] = make(map[string]common.Listener, 0)
		}

		newMap[l.Kind()][l.Name()] = l
	}

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

	e.Listeners = result
}

// Notify notifies emitter with webhook
func (e *Emitter) Notify(event testkube.Event) {
	event.ClusterName = e.ClusterName
	event.Envs = e.Envs
	err := e.Bus.PublishTopic(event.Topic(), event)
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
	err := e.Bus.SubscribeTopic("events.>", l.Name(), e.notifyHandler(l))
	if err != nil {
		e.Log.Errorw("error subscribing to event", "error", err)
	}
}

func (e *Emitter) stopListener(name string) {
	e.Log.Infow("stoping listener", name)
	err := e.Bus.Unsubscribe(name)
	if err != nil {
		e.Log.Errorw("error unsubscribing from event", "error", err)
	}
}

func (e *Emitter) notifyHandler(l common.Listener) bus.Handler {
	log := e.Log.With("listen-on", l.Events(), "queue-group", l.Name(), "selector", l.Selector(), "metadata", l.Metadata())
	return func(event testkube.Event) error {
		if event.Valid(l.Selector(), l.Events()) {
			log.Infow("notification result", l.Notify(event))
			log.Infow("listener notified", event.Log()...)
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
			e.Log.Debugw("reconciled listeners", e.Logs()...)
			time.Sleep(reconcileInterval)
		}
	}
}

func (e *Emitter) Logs() []any {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.Listeners.Log()
}

func (e *Emitter) GetListeners() common.Listeners {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.Listeners
}
