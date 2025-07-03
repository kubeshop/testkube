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
func NewEmitter(eventBus bus.Bus, clusterName string) *Emitter {
	return &Emitter{
		Log:         log.DefaultLogger,
		Loader:      NewLoader(),
		Bus:         eventBus,
		Listeners:   make(common.Listeners, 0),
		ClusterName: clusterName,
	}
}

type Interface interface {
	Notify(event testkube.Event)
}

// Emitter handles events emitting for webhooks
type Emitter struct {
	Listeners   common.Listeners
	Loader      *Loader
	Log         *zap.SugaredLogger
	mutex       sync.RWMutex
	Bus         bus.Bus
	ClusterName string
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

	result := make([]common.Listener, 0)

	oldMap := listerersToMap(e.Listeners)
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

	e.Listeners = result
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
	event.ClusterName = e.ClusterName
	err := e.Bus.PublishTopic(event.Topic(), event)
	if err != nil {
		e.Log.Errorw("error publishing event", append(event.Log(), "error", err))
		return
	}
	e.Log.Debugw("event published", event.Log()...)
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
		// NOTE: starts a listener routine for each loaded listener
		go e.startListener(l)
	}
}

func (e *Emitter) startListener(l common.Listener) {
	// NOTE: this is the listener topic for listeners which listen on the agentevents subjects
	// TODO: what is publishing to this subject?
	err := e.Bus.SubscribeTopic("agentevents.>", l.Name(), e.notifyHandler(l))
	if err != nil {
		e.Log.Errorw("error while starting listener", "error", err)
	}
	e.Log.Infow("started listener", l.Name(), l.Metadata())
}

func (e *Emitter) stopListener(name string) {
	err := e.Bus.Unsubscribe(name)
	if err != nil {
		e.Log.Errorw("error while stopping listener", "error", err)
	}
	e.Log.Info("stopped listener", name)
}

func (e *Emitter) notifyHandler(l common.Listener) bus.Handler {
	// NOTE: this is where the events are handled
	logger := e.Log.With("listen-on", l.Events(), "queue-group", l.Name(), "selector", l.Selector(), "metadata", l.Metadata())
	return func(event testkube.Event) error {
		// NOTE: seems to do some filtering of the events
		if types, valid := event.Valid(l.Selector(), l.Events()); valid {
			for i := range types {
				// TODO: wahat are these types?
				event.Type_ = &types[i]
				// NOTE: notifies the listener here
				result := l.Notify(event)
				log.Tracew(logger, "listener notified", append(event.Log(), "result", result)...)
			}
		} else {
			log.Tracew(logger, "dropping event not matching selector or type", event.Log()...)
		}
		return nil
	}
}

// Reconcile reloads listeners from all registered reconcilers
func (e *Emitter) Reconcile(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			e.Log.Info("stopping reconciler")
			return
		default:
			listeners := e.Loader.Reconcile()
			e.UpdateListeners(listeners)
			log.Tracew(e.Log, "reconciled listeners", e.Logs()...)
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
