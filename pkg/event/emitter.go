package event

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
)

const (
	eventsBuffer      = 10000
	workersCount      = 20
	reconcileInterval = time.Second
)

// NewEmitter returns new emitter instance
func NewEmitter() *Emitter {
	return &Emitter{
		Events:  make(chan testkube.Event, eventsBuffer),
		Results: make(chan testkube.EventResult, eventsBuffer),
		Log:     log.DefaultLogger,
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
}

// Register adds new listener
func (e *Emitter) Register(listener common.Listener) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.Listeners = append(e.Listeners, listener)
}

// Notify notifies emitter with webhook
func (e *Emitter) OverrideListeners(listeners common.Listeners) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.Listeners = listeners
}

// Notify notifies emitter with webhook
func (e *Emitter) Notify(event testkube.Event) {
	e.Events <- event
}

// RunWorkers runs emitter workers responsible for sending HTTP requests
func (e *Emitter) RunWorkers() {
	e.Log.Debugw("Starting event emitter workers", "count", workersCount)
	for i := 0; i < workersCount; i++ {
		go e.RunWorker(e.Events, e.Results)
	}
}

// RunWorker runs single emitter worker loop responsible for sending events
func (e *Emitter) RunWorker(events chan testkube.Event, results chan testkube.EventResult) {
	// TODO consider scaling this part to goroutines - for now we can just scale workers
	for event := range events {
		e.Log.Infow("processing event", event.Log()...)
		for _, listener := range e.Listeners {
			if event.Valid(listener.Selector()) {
				e.Log.Infow("processing event by listener", "metadata", listener.Metadata(), "selector", listener.Selector(), "kind", listener.Kind())
				results <- listener.Notify(event)
			}
		}
	}
}

// Reconcile reloads listeners from all registered reconcilers
func (s *Emitter) Reconcile(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.Log.Infow("stopping watcher")
			return
		default:
			listeners := s.Loader.Reconcile()
			s.OverrideListeners(listeners)
			s.Log.Debugw("reconciled listeners", s.Listeners.Log()...)
			time.Sleep(reconcileInterval)
		}
	}
}
