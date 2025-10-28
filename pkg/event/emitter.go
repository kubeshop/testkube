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
		loader:       NewLoader(),
		log:          log.DefaultLogger.With("instance_id", instanceId),
		bus:          eventBus,
		instanceId:   instanceId,
		leaseBackend: leaseBackend,
		subjectRoot:  subjectRoot,
		listeners:    make(common.Listeners, 0),
		clusterName:  clusterName,
	}
}

type Interface interface {
	Notify(event testkube.Event)
}

// Emitter handles events emitting for webhooks
type Emitter struct {
	*loader

	log          *zap.SugaredLogger
	listeners    common.Listeners
	mutex        sync.RWMutex
	bus          bus.Bus
	instanceId   string
	leaseBackend leasebackend.Repository
	subjectRoot  string
	clusterName  string
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
	err := e.bus.PublishTopic(e.eventTopic(event), event)
	if err != nil {
		e.log.Errorw("error publishing event", append(event.Log(), "error", err))
		return
	}
	e.log.Debugw("event published", event.Log()...)
}

// eventTopic returns topic to publish a particular evnet.
func (e *Emitter) eventTopic(event testkube.Event) string {
	// TODO(emil): is even necessary only used in tests, it does not makes sense to allow an override here considering where we are subscribed to
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

// TODO(emil): integrate this into listen
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
			e.log.Error("event emitter stopping")
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

	// Recocile loop
	go e.leaderReconcileLoop(ctx)

	// Subscribe and handle events
	e.log.Infow("event emitter leader subscribing to events")
	err := e.bus.SubscribeTopic(e.subjectRoot+".>", eventEmitterQueueName, e.eventHandler)
	if err != nil {
		e.log.Errorw("error while starting to listen for events", "error", err)
	}
	e.log.Infow("event emitter leader subscribed to events")
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
		// NOTE(emil): This fanout behavior is old, but kept intact because it
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

// leaderReconcileLoop reloads listeners from all registered reconcilers should
// only be ran by the leader.
func (e *Emitter) leaderReconcileLoop(ctx context.Context) {
	e.log.Info("event emitter leader started reconciler")
	ticker := time.NewTicker(reconcileInterval)
	for {
		select {
		case <-ctx.Done():
			e.log.Info("event emitter leader stopped reconciler")
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
