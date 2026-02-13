package event

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
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
	defaultEventTTL              = 24 * time.Hour
)

// NewEmitter returns new emitter instance
func NewEmitter(eventBus bus.Bus, leaseBackend leasebackend.Repository, subjectRoot string, clusterName string) *Emitter {
	instanceId := utils.RandAlphanum(10)
	cache := ttlcache.New[string, bool](
		ttlcache.WithTTL[string, bool](defaultEventTTL),
	)
	go cache.Start()
	return &Emitter{
		loader:              NewLoader(),
		log:                 log.DefaultLogger.With("instance_id", instanceId),
		bus:                 eventBus,
		instanceId:          instanceId,
		leaseBackend:        leaseBackend,
		subjectRoot:         subjectRoot,
		registeredListeners: make(common.Listeners, 0),
		listeners:           make(common.Listeners, 0),
		clusterName:         clusterName,
		eventCache:          cache,
		eventTTL:            defaultEventTTL,
	}
}

type Interface interface {
	Notify(event testkube.Event)
}

// Emitter handles events emitting for webhooks
type Emitter struct {
	*loader

	log                 *zap.SugaredLogger
	registeredListeners common.Listeners
	listeners           common.Listeners
	mutex               sync.RWMutex
	bus                 bus.Bus
	instanceId          string
	leaseBackend        leasebackend.Repository
	subjectRoot         string
	clusterName         string
	eventCache          *ttlcache.Cache[string, bool]
	eventTTL            time.Duration
}

// uniqueListeners keeps a unique set of listeners by kind, group and name.
// The last listener for each kind, group and name combination takes precedence.
func uniqueListeners(listeners []common.Listener) []common.Listener {
	set := make(map[string]struct{})
	unique := make(common.Listeners, 0, len(listeners))
	for _, listener := range slices.Backward(listeners) {
		key := listener.Kind() + "/" + listener.Group() + "/" + listener.Name()
		if _, exists := set[key]; exists {
			continue
		}
		set[key] = struct{}{}
		unique = append(unique, listener)
	}
	return unique
}

func (e *Emitter) registerListener(listener common.Listener) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.registeredListeners = uniqueListeners(append(e.registeredListeners, listener))
	e.listeners = uniqueListeners(append(e.listeners, e.registeredListeners...))
}

func (e *Emitter) loadListeners(loadedListeners []common.Listener) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.listeners = uniqueListeners(append(loadedListeners, e.registeredListeners...))
}

// Register adds new listener
func (e *Emitter) Register(listener common.Listener) {
	e.registerListener(listener)
}

// Notify notifies emitter with webhook
func (e *Emitter) Notify(event testkube.Event) {
	event.ClusterName = e.clusterName
	topic := e.eventTopic(event)

	eventType := "unknown"
	if event.Type_ != nil {
		eventType = string(*event.Type_)
	}

	e.log.Debugw("publishing event",
		"event_type", eventType,
		"topic", topic,
		"resource", event.Resource,
		"resource_id", event.ResourceId)

	err := e.bus.PublishTopic(topic, event)
	if err != nil {
		e.log.Errorw("failed to publish event",
			"event_type", eventType,
			"topic", topic,
			"error", err)
		return
	}
	e.log.Debugw("event published successfully",
		"event_type", eventType,
		"topic", topic)
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

// TODO(emil): convert to using new common coordinator package for lease acquisition
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
		e.log.Info("event emitter stopping event cache")
		e.eventCache.Stop()
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
				e.log.Debug("event emitter no longer leader stop listening")
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
	e.loadListeners(e.Reconcile())
	log.Tracew(e.log, "reconciled listeners", e.ListenersDump()...)
	// Subscribe and handle events
	subscribeSubject := e.subjectRoot + ".>"
	e.log.Infow("event emitter leader subscribing to events",
		"subject", subscribeSubject,
		"queue_name", eventEmitterQueueName)
	err := e.bus.SubscribeTopic(subscribeSubject, eventEmitterQueueName, e.leaderEventHandler)
	if err != nil {
		e.log.Errorw("error while starting to listen for events",
			"subject", subscribeSubject,
			"error", err)
	}
	e.log.Infow("event emitter leader subscribed to events",
		"subject", subscribeSubject)
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
			e.loadListeners(e.Reconcile())
			log.Tracew(e.log, "reconciled listeners", e.ListenersDump()...)
		}
	}
}

func (e *Emitter) leaderEventHandler(event testkube.Event) error {
	// Log event received by leader
	eventType := "unknown"
	if event.Type_ != nil {
		eventType = string(*event.Type_)
	}
	e.log.Debugw("leader received event",
		"event_type", eventType,
		"resource", event.Resource,
		"resource_id", event.ResourceId,
		"event_groupid", event.GroupId,
		"event_id", event.Id)

	// Check for duplicate event using event.Id for idempotency
	if event.Id != "" {
		if e.eventCache.Has(event.Id) {
			e.log.Debugw("skipping duplicate event",
				"event_id", event.Id,
				"event_type", eventType)
			return nil
		}
		// Store event.Id in cache to prevent duplicate processing
		e.eventCache.Set(event.Id, true, ttlcache.DefaultTTL)
	}

	// Current set of listeners
	e.mutex.Lock()
	listenersSnapshot := make([]common.Listener, len(e.listeners))
	copy(listenersSnapshot, e.listeners)
	e.mutex.Unlock()

	// Track matched listeners
	matchedCount := 0

	// Find listeners that match the event
	for _, l := range listenersSnapshot {
		logger := e.log.With("listen-on", l.Events(), "selector", l.Selector(), "metadata", l.Metadata())
		if !l.Match(event) {
			log.Tracew(logger, "dropping event not matching selector or type", event.Log()...)
			continue
		}
		// Event type fanout
		matchedEventTypes, valid := event.Valid(l.Group(), l.Selector(), l.Events())
		if !valid {
			e.log.Debugw("listener did not match event",
				"listener_name", l.Name(),
				"listener_kind", l.Kind(),
				"listener_group", l.Group(),
				"event_groupid", event.GroupId,
				"matched", false)
			continue
		}
		for i := range matchedEventTypes {
			event.Type_ = &matchedEventTypes[i]
			matchedCount++
			e.log.Debugw("notifying listener",
				"listener_name", l.Name(),
				"listener_kind", l.Kind(),
				"event_type", string(matchedEventTypes[i]))
			go notifyListener(logger, l, event)
		}
	}

	if matchedCount == 0 {
		e.log.Debugw("no listeners matched event",
			"event_type", eventType,
			"total_listeners", len(listenersSnapshot))
	} else {
		e.log.Infow("event matched listeners",
			"event_type", eventType,
			"matched_count", matchedCount,
			"total_listeners", len(listenersSnapshot))
	}

	return nil
}

func notifyListener(logger *zap.SugaredLogger, listener common.Listener, event testkube.Event) {
	// TODO(emil): Held over from old implementation. These results are just
	// logged, not sure there is much point even returning them, can just log
	// in the listener and all this can be simplified to use Notify.
	eventType := "unknown"
	if event.Type_ != nil {
		eventType = string(*event.Type_)
	}

	result := listener.Notify(event)

	if result.Error() != "" {
		logger.Errorw("listener notification failed",
			"listener_name", listener.Name(),
			"listener_kind", listener.Kind(),
			"event_type", eventType,
			"error", result.Error())
	}
}

// ListenersDump dumps all the currently reconciled listeners in an array for debugging.
func (e *Emitter) ListenersDump() []any {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.listeners.Dump()
}
