package event

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/event/kind/dummy"
	"github.com/kubeshop/testkube/pkg/event/kind/k8sevent"
	"github.com/kubeshop/testkube/pkg/repository/leasebackend"
)

func init() {
	os.Setenv("DEBUG", "true")
}

// getListeners allows getting listeners in a multithreaded fashion only used by tests.
func (e *Emitter) getListeners() common.Listeners {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.listeners
}

func TestEmitter_Register(t *testing.T) {
	t.Run("Register adds new listener", func(t *testing.T) {
		// given
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()
		// when
		emitter.Register(&dummy.DummyListener{Id: "l1"})

		// then
		assert.Equal(t, 1, len(emitter.getListeners()))

		t.Log("T1 completed")
	})
}

func TestEmitter_Listen(t *testing.T) {
	t.Run("listener handles only given events based on selectors", func(t *testing.T) {
		// given
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		mockLeaseRepository.EXPECT().
			TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, nil).AnyTimes()
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()
		// given listener with matching selector
		listener1 := &dummy.DummyListener{Id: "l1", SelectorString: "type=listener1"}
		// and listener with second matching selector
		listener2 := &dummy.DummyListener{Id: "l2", SelectorString: "type=listener2"}

		// and emitter with registered listeners
		emitter.Register(listener1)
		emitter.Register(listener2)

		// listening emitter
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go emitter.Listen(ctx)
		// wait for listeners to start
		time.Sleep(50 * time.Millisecond)

		// events
		event1 := newExampleTestEvent1()
		event1.TestWorkflowExecution.Workflow.Labels = map[string]string{"type": "listener1"}
		event2 := newExampleTestEvent2()
		event2.TestWorkflowExecution.Workflow.Labels = map[string]string{"type": "listener2"}

		// when
		emitter.Notify(event1)
		emitter.Notify(event2)

		// then
		retryCount := 100
		notificationsCountListener1 := 0
		notificationsCountListener2 := 0
		for i := 0; i < retryCount; i++ {
			notificationsCountListener1 = listener1.GetNotificationCount()
			notificationsCountListener2 = listener2.GetNotificationCount()
			if notificationsCountListener1 == 1 && notificationsCountListener2 == 1 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		assert.Equal(t, 1, notificationsCountListener1)
		assert.Equal(t, 1, notificationsCountListener2)
	})
}

func TestEmitter_Notify(t *testing.T) {
	t.Run("notifies listeners in queue groups", func(t *testing.T) {
		// given
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		mockLeaseRepository.EXPECT().
			TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, nil).AnyTimes()
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()
		// and 2 listeners subscribed to the same queue
		// * first on pod1
		listener1 := &dummy.DummyListener{Id: "l3"}
		// * second on pod2
		listener2 := &dummy.DummyListener{Id: "l3"}

		emitter.Register(listener1)
		emitter.Register(listener2)

		// and listening emitter
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go emitter.Listen(ctx)

		time.Sleep(time.Millisecond * 50)

		// when event sent to queue group
		emitter.Notify(newExampleTestEvent1())

		time.Sleep(time.Millisecond * 50)

		// then only one listener should be notified
		assert.Equal(t, 1, listener2.GetNotificationCount()+listener1.GetNotificationCount())
	})
}

func TestEmitter_NotifyBecome(t *testing.T) {
	t.Run("notifies listeners in queue groups for become events", func(t *testing.T) {
		// given
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		mockLeaseRepository.EXPECT().
			TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, nil).AnyTimes()
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()
		// and 2 listeners subscribed to the same queue
		// * first on pod1
		listener1 := &dummy.DummyListener{Id: "l5", Types: []testkube.EventType{
			testkube.BECOME_TESTWORKFLOW_FAILED_EventType, testkube.BECOME_TESTWORKFLOW_DOWN_EventType, testkube.END_TESTWORKFLOW_FAILED_EventType,
		}}
		// * second on pod2
		listener2 := &dummy.DummyListener{Id: "l5", Types: []testkube.EventType{
			testkube.BECOME_TESTWORKFLOW_FAILED_EventType, testkube.BECOME_TESTWORKFLOW_DOWN_EventType, testkube.END_TESTWORKFLOW_FAILED_EventType,
		}}

		emitter.Register(listener1)
		emitter.Register(listener2)

		// and listening emitter
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go emitter.Listen(ctx)

		time.Sleep(time.Millisecond * 50)

		// when event sent to queue group
		emitter.Notify(newExampleTestEvent5())

		time.Sleep(time.Millisecond * 50)

		// then only one listener should be notified
		assert.Equal(t, 3, listener2.GetNotificationCount()+listener1.GetNotificationCount())
	})
}

func TestEmitter_Listen_reconciliation(t *testing.T) {
	t.Run("emitter refresh listeners in reconcile loop", func(t *testing.T) {
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		mockLeaseRepository.EXPECT().
			TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, nil).AnyTimes()
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()
		emitter.RegisterLoader(&dummy.DummyLoader{IdPrefix: "dummy1"})
		emitter.RegisterLoader(&dummy.DummyLoader{IdPrefix: "dummy2"})

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go emitter.Listen(ctx)
		time.Sleep(50 * time.Millisecond)

		assert.Len(t, emitter.getListeners(), 4)

		// and we'll add additional new loader
		emitter.RegisterLoader(&dummy.DummyLoader{IdPrefix: "dummy1"}) // should be ignored
		emitter.RegisterLoader(&dummy.DummyLoader{IdPrefix: "dummy3"})

		// then each reconciler (3 reconcilers) should load 2 listeners
		time.Sleep(1100 * time.Millisecond)
		assert.Len(t, emitter.getListeners(), 6)
	})

	t.Run("emitter updates listeners in reconcile loop", func(t *testing.T) {
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		mockLeaseRepository.EXPECT().
			TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, nil).AnyTimes()
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()
		registeredListener := &dummy.DummyListener{Id: "registered", Types: []testkube.EventType{
			testkube.BECOME_TESTWORKFLOW_UP_EventType,
		}}
		emitter.Register(registeredListener)
		emitter.RegisterLoader(&dummy.DummyLoader{IdPrefix: "dummy1", SelectorString: "v1"})

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go emitter.Listen(ctx)
		time.Sleep(50 * time.Millisecond)

		assert.Len(t, emitter.getListeners(), 3)
		assert.Equal(t, "v1", emitter.getListeners()[1].Selector())

		// This loader should overwrite the items loaded from the first loader
		// on next reconiliation loop
		emitter.RegisterLoader(&dummy.DummyLoader{IdPrefix: "dummy1", SelectorString: "v2"})

		time.Sleep(1100 * time.Millisecond)
		assert.Len(t, emitter.getListeners(), 3)
		assert.Equal(t, "v2", emitter.getListeners()[1].Selector())
	})

	t.Run("emitter remove listeners in reconcile loop", func(t *testing.T) {
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		mockLeaseRepository.EXPECT().
			TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, nil).AnyTimes()
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()
		loader := &dummy.DummyLoader{IdPrefix: "dummy1", SelectorString: "v1"}
		registeredListener := &dummy.DummyListener{Id: "registered", Types: []testkube.EventType{
			testkube.BECOME_TESTWORKFLOW_UP_EventType,
		}}
		emitter.Register(registeredListener)
		emitter.RegisterLoader(loader)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go emitter.Listen(ctx)
		time.Sleep(50 * time.Millisecond)

		assert.Len(t, emitter.getListeners(), 3)
		assert.Equal(t, "v1", emitter.getListeners()[1].Selector())

		// Override the listeners in the loader to return an empty set of
		// listeners to test deletion
		loader.ListenersOverride = []common.Listener{}

		// Wait to next reconcillation loop
		time.Sleep(1100 * time.Millisecond)
		// Only the registered listener should remain
		assert.Len(t, emitter.getListeners(), 1)
		assert.Equal(t, "registered", emitter.getListeners()[0].Name())
	})
}

func TestEmitterCreatesK8sEvents(t *testing.T) {
	t.Parallel()

	eventBus := bus.NewEventBusMock()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
	mockLeaseRepository.EXPECT().
		TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(true, nil).AnyTimes()

	clientset := fake.NewSimpleClientset()
	listener := k8sevent.NewK8sEventListener("k8sevent", "", "tk-dev",
		[]testkube.EventType{*testkube.EventStartTestWorkflow}, clientset)

	emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
	defer emitter.eventCache.Stop()
	emitter.Register(listener)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go emitter.Listen(ctx)
	time.Sleep(50 * time.Millisecond)

	event := testkube.Event{
		Id:                    "event-k8s",
		Type_:                 testkube.EventStartTestWorkflow,
		TestWorkflowExecution: testkube.NewExecutionWithID("exec-k8s", "workflow-k8s"),
	}

	emitter.Notify(event)

	var gotEvent bool
	for i := 0; i < 20; i++ {
		evts, err := clientset.CoreV1().Events("tk-dev").List(context.Background(), metav1.ListOptions{})
		assert.NoError(t, err)
		if len(evts.Items) > 0 {
			gotEvent = true
			assert.Equal(t, "testkube-event-event-k8s", evts.Items[0].Name)
			assert.Equal(t, "start-testworkflow", evts.Items[0].Reason)
			assert.Equal(t, "started", evts.Items[0].Action)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.True(t, gotEvent, "expected a Kubernetes event to be created")
}

func newExampleTestEvent1() testkube.Event {
	return testkube.Event{
		Id:                    "eventID1",
		Type_:                 testkube.EventStartTestWorkflow,
		TestWorkflowExecution: testkube.NewExecutionWithID("executionID1", "test"),
	}
}

func newExampleTestEvent2() testkube.Event {
	return testkube.Event{
		Id:                    "eventID2",
		Type_:                 testkube.EventStartTestWorkflow,
		TestWorkflowExecution: testkube.NewExecutionWithID("executionID2", "test"),
	}
}

func newExampleTestEvent5() testkube.Event {
	return testkube.Event{
		Id:                    "eventID5",
		Type_:                 testkube.EventEndTestWorkflowFailed,
		TestWorkflowExecution: testkube.NewExecutionWithID("executionID5", "test"),
	}
}

var _ common.Listener = (*FakeListener)(nil)

type FakeListener struct {
	name string
}

func (l *FakeListener) Match(event testkube.Event) bool {
	return true
}

func (l *FakeListener) Notify(event testkube.Event) testkube.EventResult {
	return testkube.EventResult{Id: event.Id}
}

func (l *FakeListener) Name() string {
	return l.name
}

func (l *FakeListener) Events() []testkube.EventType {
	return nil
}

func (l FakeListener) Selector() string {
	return ""
}

func (l *FakeListener) Kind() string {
	return "fake"
}

func (l *FakeListener) Group() string {
	return ""
}

func (l *FakeListener) Metadata() map[string]string {
	return map[string]string{}
}

func TestEmitter_eventTopic(t *testing.T) {
	emitter := NewEmitter(nil, nil, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)

	t.Run("should return events topic if explicitly set", func(t *testing.T) {
		evt := testkube.Event{Type_: testkube.EventEndTestWorkflowSuccess, StreamTopic: "topic"}
		assert.Equal(t, "topic", emitter.eventTopic(evt))
	})

	t.Run("should return events topic if not resource set", func(t *testing.T) {
		evt := testkube.Event{Type_: testkube.EventEndTestWorkflowSuccess, Resource: nil}
		assert.Equal(t, "agentevents.all", emitter.eventTopic(evt))
	})

	t.Run("should return event topic with resource name and id if set", func(t *testing.T) {
		evt := testkube.Event{
			Type_:    testkube.EventEndTestWorkflowSuccess,
			Resource: testkube.EventResourceTestWorkflowExecution, ResourceId: "123",
		}
		assert.Equal(t, "agentevents.testworkflowexecution.123", emitter.eventTopic(evt))
	})

	t.Run("should return event topic with resource name when id not set", func(t *testing.T) {
		evt := testkube.Event{
			Type_:    testkube.EventEndTestWorkflowSuccess,
			Resource: testkube.EventResourceTestWorkflowExecution,
		}
		assert.Equal(t, "agentevents.testworkflowexecution", emitter.eventTopic(evt))
	})
}

func TestEmitter_Idempotency(t *testing.T) {
	t.Run("should process event with same ID only once", func(t *testing.T) {
		// given
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		mockLeaseRepository.EXPECT().
			TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, nil).AnyTimes()
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()

		listener := &dummy.DummyListener{Id: "l1"}
		emitter.Register(listener)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go emitter.Listen(ctx)
		time.Sleep(50 * time.Millisecond)

		// when - sending the same event twice with the same ID
		event := newExampleTestEvent1()
		emitter.Notify(event)

		// then - wait for first event to be processed
		retryCount := 100
		for i := 0; i < retryCount; i++ {
			if listener.GetNotificationCount() >= 1 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		// Store the notification count after first event
		firstCount := listener.GetNotificationCount()
		assert.Equal(t, 1, firstCount, "listener should have been notified once after first event")

		// Send the same event again (duplicate)
		emitter.Notify(event)
		time.Sleep(100 * time.Millisecond)

		// then - listener should still be notified only once
		assert.Equal(t, 1, listener.GetNotificationCount(), "listener should still have been notified only once after duplicate event")
	})

	t.Run("should process events with different IDs", func(t *testing.T) {
		// given
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		mockLeaseRepository.EXPECT().
			TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, nil).AnyTimes()
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()

		listener := &dummy.DummyListener{Id: "l1"}
		emitter.Register(listener)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go emitter.Listen(ctx)
		time.Sleep(50 * time.Millisecond)

		// when - sending two different events
		event1 := newExampleTestEvent1()
		event2 := newExampleTestEvent2()

		emitter.Notify(event1)
		emitter.Notify(event2)

		// then - wait for notifications
		retryCount := 100
		notificationsCount := 0
		for i := 0; i < retryCount; i++ {
			notificationsCount = listener.GetNotificationCount()
			if notificationsCount == 2 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		assert.Equal(t, 2, notificationsCount, "listener should have been notified twice for two different events")
	})

	t.Run("should process event without ID normally", func(t *testing.T) {
		// given
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		mockLeaseRepository.EXPECT().
			TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, nil).AnyTimes()
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()

		listener := &dummy.DummyListener{Id: "l1"}
		emitter.Register(listener)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go emitter.Listen(ctx)
		time.Sleep(50 * time.Millisecond)

		// when - sending event without ID (idempotency check should be skipped)
		event := testkube.Event{
			Id:                    "", // No ID
			Type_:                 testkube.EventStartTestWorkflow,
			TestWorkflowExecution: testkube.NewExecutionWithID("executionID3", "test"),
		}

		emitter.Notify(event)
		emitter.Notify(event) // Send again

		// then - both should be processed
		retryCount := 100
		notificationsCount := 0
		for i := 0; i < retryCount; i++ {
			notificationsCount = listener.GetNotificationCount()
			if notificationsCount == 2 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		assert.Equal(t, 2, notificationsCount, "listener should have been notified twice when event has no ID")
	})
}

func TestEmitter_MultipleWebhooksWithDifferentEventTypes(t *testing.T) {
	t.Run("should notify all webhooks with correct event types when multiple webhooks match", func(t *testing.T) {
		// This test reproduces the issue from https://github.com/kubeshop/testkube/issues/7055
		// where webhooks don't send events when multiple webhooks with different event types are configured

		// given
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		mockLeaseRepository.EXPECT().
			TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, nil).AnyTimes()
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()

		// Create listeners that match the issue scenario:
		// 1. Webhook for become-testworkflow-up (matches with selector)
		becomeUpListener := &dummy.DummyListener{
			Id: "become-up-webhook",
			Types: []testkube.EventType{
				testkube.BECOME_TESTWORKFLOW_UP_EventType,
			},
			SelectorString: "ataccama.com/incident-policy=warning-when-state-change",
		}

		// 2. Webhook for end-testworkflow-success (no selector, matches all)
		endSuccessListener := &dummy.DummyListener{
			Id: "otel-webhook",
			Types: []testkube.EventType{
				testkube.END_TESTWORKFLOW_SUCCESS_EventType,
			},
			SelectorString: "",
		}

		emitter.Register(becomeUpListener)
		emitter.Register(endSuccessListener)

		// Start listening
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go emitter.Listen(ctx)
		time.Sleep(50 * time.Millisecond)

		// Create an event for a successful test workflow with the matching label
		event := testkube.Event{
			Id:    "event-multi-webhook",
			Type_: testkube.EventEndTestWorkflowSuccess,
			TestWorkflowExecution: &testkube.TestWorkflowExecution{
				Id: "exec-multi",
				Workflow: &testkube.TestWorkflow{
					Name: "tw1",
					Labels: map[string]string{
						"ataccama.com/incident-policy": "warning-when-state-change",
					},
				},
			},
		}

		// when
		emitter.Notify(event)

		// then - wait for notifications
		time.Sleep(100 * time.Millisecond)

		// Both listeners should have been notified
		becomeUpReceivedTypes := becomeUpListener.GetReceivedEventTypes()
		endSuccessReceivedTypes := endSuccessListener.GetReceivedEventTypes()

		// The become-up listener should receive the BECOME_TESTWORKFLOW_UP event
		assert.Contains(t, becomeUpReceivedTypes, testkube.BECOME_TESTWORKFLOW_UP_EventType,
			"become-up webhook should receive BECOME_TESTWORKFLOW_UP event")

		// The end-success listener should receive the END_TESTWORKFLOW_SUCCESS event
		assert.Contains(t, endSuccessReceivedTypes, testkube.END_TESTWORKFLOW_SUCCESS_EventType,
			"otel webhook should receive END_TESTWORKFLOW_SUCCESS event")

		// Verify each listener received exactly the event type it subscribed to
		assert.Len(t, becomeUpReceivedTypes, 1, "become-up webhook should receive exactly 1 event")
		assert.Len(t, endSuccessReceivedTypes, 1, "otel webhook should receive exactly 1 event")
	})
}

func TestEmitter_ThreeWebhooksRealScenario(t *testing.T) {
	t.Run("should correctly handle three webhooks: failed, up, and otel with end-testworkflow-success event", func(t *testing.T) {
		// This test reproduces the exact scenario from the issue report
		// where three webhooks are configured and an end-testworkflow-success event fires

		// given
		eventBus := bus.NewEventBusMock()
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockLeaseRepository := leasebackend.NewMockRepository(mockCtrl)
		mockLeaseRepository.EXPECT().
			TryAcquire(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(true, nil).AnyTimes()
		emitter := NewEmitter(eventBus, mockLeaseRepository, "agentevents", "", DefaultEventTTL, DefaultEventCacheCapacity)
		defer emitter.eventCache.Stop()

		// Webhook 1: "failed" - become-testworkflow-failed with selector
		failedWebhook := &dummy.DummyListener{
			Id: "failed",
			Types: []testkube.EventType{
				testkube.BECOME_TESTWORKFLOW_FAILED_EventType,
			},
			SelectorString: "ataccama.com/incident-policy=warning-when-state-change",
		}

		// Webhook 2: "up" - become-testworkflow-up with selector
		upWebhook := &dummy.DummyListener{
			Id: "up",
			Types: []testkube.EventType{
				testkube.BECOME_TESTWORKFLOW_UP_EventType,
			},
			SelectorString: "ataccama.com/incident-policy=warning-when-state-change",
		}

		// Webhook 3: "testkube-otel-webhook" - multiple events, no selector
		otelWebhook := &dummy.DummyListener{
			Id: "testkube-otel-webhook",
			Types: []testkube.EventType{
				testkube.START_TESTWORKFLOW_EventType,
				testkube.END_TESTWORKFLOW_SUCCESS_EventType,
				testkube.END_TESTWORKFLOW_FAILED_EventType,
				testkube.END_TESTWORKFLOW_ABORTED_EventType,
			},
			SelectorString: "", // No selector - matches all
		}

		emitter.Register(failedWebhook)
		emitter.Register(upWebhook)
		emitter.Register(otelWebhook)

		// listening emitter
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go emitter.Listen(ctx)
		time.Sleep(50 * time.Millisecond)

		// when: an end-testworkflow-success event fires with the matching label
		event := testkube.Event{
			Id:    "test-event-success",
			Type_: testkube.EventEndTestWorkflowSuccess,
			TestWorkflowExecution: &testkube.TestWorkflowExecution{
				Id: "exec-success-1",
				Workflow: &testkube.TestWorkflow{
					Name: "test-workflow",
					Labels: map[string]string{
						"ataccama.com/incident-policy": "warning-when-state-change",
					},
				},
			},
		}

		emitter.Notify(event)

		// then - wait for notifications
		time.Sleep(100 * time.Millisecond)

		// Get received event types
		failedReceivedTypes := failedWebhook.GetReceivedEventTypes()
		upReceivedTypes := upWebhook.GetReceivedEventTypes()
		otelReceivedTypes := otelWebhook.GetReceivedEventTypes()

		t.Logf("failed webhook received: %v (count: %d)", failedReceivedTypes, len(failedReceivedTypes))
		t.Logf("up webhook received: %v (count: %d)", upReceivedTypes, len(upReceivedTypes))
		t.Logf("otel webhook received: %v (count: %d)", otelReceivedTypes, len(otelReceivedTypes))

		// Verify expectations:
		// 1. "failed" webhook should NOT receive any event (become-testworkflow-failed doesn't map to success)
		assert.Empty(t, failedReceivedTypes,
			"failed webhook should NOT receive any event for end-testworkflow-success")

		// 2. "up" webhook SHOULD receive become-testworkflow-up (maps to end-testworkflow-success)
		assert.Contains(t, upReceivedTypes, testkube.BECOME_TESTWORKFLOW_UP_EventType,
			"up webhook should receive BECOME_TESTWORKFLOW_UP event")
		assert.Len(t, upReceivedTypes, 1,
			"up webhook should receive exactly 1 event")

		// 3. "otel" webhook SHOULD receive end-testworkflow-success (direct match, no selector)
		assert.Contains(t, otelReceivedTypes, testkube.END_TESTWORKFLOW_SUCCESS_EventType,
			"otel webhook should receive END_TESTWORKFLOW_SUCCESS event")
		assert.Len(t, otelReceivedTypes, 1,
			"otel webhook should receive exactly 1 event")

		// Total: 2 webhooks should be notified (up + otel)
		totalNotifications := len(failedReceivedTypes) + len(upReceivedTypes) + len(otelReceivedTypes)
		assert.Equal(t, 2, totalNotifications,
			"Total notifications should be 2 (up webhook + otel webhook)")
	})
}
