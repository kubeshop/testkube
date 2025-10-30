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
	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
	"github.com/kubeshop/testkube/pkg/utils"
)

const (
	reconcileInterval            = time.Second
	eventEmitterQueueName string = "emitter"
)

// NewEmitter returns new emitter instance
func NewEmitter(eventBus bus.Bus, leaseBackend leasebackend.Repository, subjectRoot string, clusterName string) *Emitter {
	instanceId := utils.RandAlphanum(10)
	return &Emitter{
		loader:         NewLoader(),
		log:            log.DefaultLogger.With("instance_id", instanceId),
		bus:            eventBus,
		instanceId:     instanceId,
		leaseBackend:   leaseBackend,
		subjectRoot:    subjectRoot,
		listeners:      make(common.Listeners, 0),
		listenerExists: make(map[string]map[string]struct{}),
		clusterName:    clusterName,
	}
}

type Interface interface {
	Notify(event testkube.Event)
}

// Emitter handles events emitting for webhooks
type Emitter struct {
	*loader

	log            *zap.SugaredLogger
	listeners      common.Listeners
	listenerExists map[string]map[string]struct{}
	mutex          sync.RWMutex
	bus            bus.Bus
	instanceId     string
	leaseBackend   leasebackend.Repository
	subjectRoot    string
	clusterName    string
}

// appendUniqueListeners appends only listeners unique by kind and name.
func (e *Emitter) appendUniqueListeners(listeners ...common.Listener) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	for i := range listeners {
		kind := listeners[i].Kind()
		name := listeners[i].Name()
		if e.listenerExists[kind] == nil {
			e.listenerExists[kind] = make(map[string]struct{})
		}
		if _, exists := e.listenerExists[kind][name]; !exists {
			// Mark as existing and append to listeners array
			e.listenerExists[kind][name] = struct{}{}
			e.listeners = append(e.listeners, listeners[i])
		}
	}
}

// Register adds new listener
func (e *Emitter) Register(listener common.Listener) {
	e.appendUniqueListeners(listener)
}

// Notify notifies emitter with webhook
func (e *Emitter) Notify(event testkube.Event) {
	event.ClusterName = e.clusterName
	err := e.bus.PublishTopic(e.eventTopic(event), event)
	if err != nil {
		e.log.Errorw("error publishing event", append(event.Log(), "error", err))
		return
	}
	e.log.Debugw("event published", event.Log()...)
}

// eventTopic returns topic to publish a particular evnet.
func (e *Emitter) eventTopic(event testkube.Event) string {
	// TODO(emil): only used in tests, it does not makes sense to allow an
	// override here because we need the topic to be prefixed a certain way for
	// our subscription to handle it
	if event.StreamTopic != "" {
		return event.StreamTopic
	}

	if event.Resource == nil {
		return e.subjectRoot + ".all"
	}

	if event.ResourceId == "" {
		return e.subjectRoot + "." + string(*event.Resource)
	}

	return e.subjectRoot + "." + string(*event.Resource) + "." + event.ResourceId
}

const (
	leaseCheckInterval        = 5 * time.Second
	leaseClusterID     string = "event-emitters"
)

func (e *Emitter) leaseCheckLoop(ctx context.Context, leaseChan chan<- bool) {
	e.log.Info("event emitter waiting for lease")
	e.leaseCheck(ctx, leaseChan)
	ticker := time.NewTicker(leaseCheckInterval)
	for {
		select {
		case <-ctx.Done():
			e.log.Info("event emitter stopped lease checks")
			return
		case <-ticker.C:
			e.leaseCheck(ctx, leaseChan)
		}
	}
}

func (e *Emitter) leaseCheck(ctx context.Context, leaseChan chan<- bool) {
	leased, err := e.leaseBackend.TryAcquire(ctx, leaseClusterID+"-"+e.instanceId, leaseClusterID)
	if err != nil {
		e.log.Errorw("error while trying to acquire lease", "error", err)
	}
	leaseChan <- leased
}

// Listen checks for lease holding and starts/stops workers processing event
// notifications.
func (e *Emitter) Listen(ctx context.Context) {
	e.log.Info("event emitter starting")
	// Clean up
	go func() {
		<-ctx.Done()
		e.log.Info("event emitter closing event bus")
		err := e.bus.Close()
		if err != nil {
			e.log.Errorw("error while closing event bus", "error", err)
		} else {
			e.log.Info("event emitter closed event bus")
		}
	}()

	// Start lease check loop
	leaseChan := make(chan bool)
	go e.leaseCheckLoop(ctx, leaseChan)

	// Current lease status
	var leaderCancel context.CancelFunc
	for {
		select {
		case <-ctx.Done():
			e.log.Info("event emitter stopping")
			if leaderCancel != nil {
				leaderCancel()
			}
			return
		case leased := <-leaseChan:
			if !leased && leaderCancel != nil {
				// Lost leadership
				e.log.Info("event emitter no longer leader stop listening")
				leaderCancel()
				leaderCancel = nil
			} else if leased && leaderCancel == nil {
				// Became leader
				e.log.Info("event emitter becoming leader")
				var leaderCtx context.Context
				leaderCtx, leaderCancel = context.WithCancel(ctx)
				go e.leaderLoop(leaderCtx)
			}
		}
	}
}

// leaderLoop reloads listeners from all registered reconcilers should
// only be ran by the leader.
// After the first reconcilation it starts subscribing and handling events.
func (e *Emitter) leaderLoop(ctx context.Context) {
	// Clean up
	go func() {
		<-ctx.Done()
		e.log.Info("event emitter leader unsubscribing from emitted events")

		err := e.bus.Unsubscribe(eventEmitterQueueName)
		if err != nil {
			e.log.Warnw("error while unsubscribing from emitted events", "error", err)
		}
		e.log.Info("event emitter leader unsubscribed from emitted events")
	}()
	// First reconcilation to avoid waiting for first tick
	listeners := e.Reconcile()
	e.appendUniqueListeners(listeners...)
	log.Tracew(e.log, "reconciled listeners", e.ListenersDump()...)
	// Subscribe and handle events
	e.log.Infow("event emitter leader subscribing to events")
	err := e.bus.SubscribeTopic(e.subjectRoot+".>", eventEmitterQueueName, e.leaderEventHandler)
	if err != nil {
		e.log.Errorw("error while starting to listen for events", "error", err)
	}
	e.log.Infow("event emitter leader subscribed to events")
	// Reconcilation loop
	e.log.Info("event emitter leader started reconciler")
	ticker := time.NewTicker(reconcileInterval)
	for {
		select {
		case <-ctx.Done():
			e.log.Info("event emitter leader stopped reconciler")
			return
		case <-ticker.C:
			// Reconcile listeners
			listeners := e.Reconcile()
			e.appendUniqueListeners(listeners...)
			log.Tracew(e.log, "reconciled listeners", e.ListenersDump()...)
		}
	}
}

func (e *Emitter) leaderEventHandler(event testkube.Event) error {
	// Current set of listeners
	e.mutex.Lock()
	listenersSnapshot := make([]common.Listener, len(e.listeners))
	copy(listenersSnapshot, e.listeners)
	e.mutex.Unlock()
	// Find listeners that match the event
	for _, l := range listenersSnapshot {
		logger := e.log.With("listen-on", l.Events(), "selector", l.Selector(), "metadata", l.Metadata())
		if !l.Match(event) {
			log.Tracew(logger, "dropping event not matching selector or type", event.Log()...)
			continue
		}
		// Event type fanout
		matchedEventTypes, _ := event.Valid(l.Selector(), l.Events())
		for i := range matchedEventTypes {
			event.Type_ = &matchedEventTypes[i]
			go notifyListener(logger, l, event)
		}
	}
	return nil
}

func notifyListener(logger *zap.SugaredLogger, listener common.Listener, event testkube.Event) {
	// TODO(emil): Held over from old implementation. These results are just
	// logged, not sure there is much point even returning them, can just log
	// in the listener and all this can be simplified to use Notify.
	result := listener.Notify(event)
	log.Tracew(logger, "listener notified", append(event.Log(), "result", result)...)
}

// ListenersDump dumps all the currently reconciled listeners in an array for debugging.
func (e *Emitter) ListenersDump() []any {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.listeners.Dump()
}
