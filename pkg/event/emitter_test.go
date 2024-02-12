package event

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/event/kind/dummy"
)

func init() {
	os.Setenv("DEBUG", "true")
}

func TestEmitter_Register(t *testing.T) {
	t.Parallel()

	t.Run("Register adds new listener", func(t *testing.T) {
		t.Parallel()
		// given
		eventBus := bus.NewEventBusMock()
		emitter := NewEmitter(eventBus, "", nil)
		// when
		emitter.Register(&dummy.DummyListener{Id: "l1"})

		// then
		assert.Equal(t, 1, len(emitter.Listeners))

		t.Log("T1 completed")
	})
}

func TestEmitter_Listen(t *testing.T) {
	t.Parallel()

	t.Run("listener handles only given events based on selectors", func(t *testing.T) {
		t.Parallel()
		// given
		eventBus := bus.NewEventBusMock()
		emitter := NewEmitter(eventBus, "", nil)
		// given listener with matching selector
		listener1 := &dummy.DummyListener{Id: "l1", SelectorString: "type=listener1"}
		// and listener with second matic selector
		listener2 := &dummy.DummyListener{Id: "l2", SelectorString: "type=listener2"}

		// and emitter with registered listeners
		emitter.Register(listener1)
		emitter.Register(listener2)

		// listening emitter
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		emitter.Listen(ctx)
		// wait for listeners to start
		time.Sleep(time.Millisecond * 50)

		// events
		event1 := newExampleTestEvent1()
		event1.TestExecution.Labels = map[string]string{"type": "listener1"}
		event2 := newExampleTestEvent2()
		event2.TestExecution.Labels = map[string]string{"type": "listener2"}

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
	t.Parallel()

	t.Run("notifies listeners in queue groups", func(t *testing.T) {
		t.Parallel()
		// given
		eventBus := bus.NewEventBusMock()
		emitter := NewEmitter(eventBus, "", nil)
		// and 2 listeners subscribed to the same queue
		// * first on pod1
		listener1 := &dummy.DummyListener{Id: "l3"}
		// * second on pod2
		listener2 := &dummy.DummyListener{Id: "l3"}

		emitter.Register(listener1)
		emitter.Register(listener2)

		// and listening emitter
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		emitter.Listen(ctx)

		time.Sleep(time.Millisecond * 50)

		// when event sent to queue group
		emitter.Notify(newExampleTestEvent1())

		time.Sleep(time.Millisecond * 50)

		// then only one listener should be notified
		assert.Equal(t, 1, listener2.GetNotificationCount()+listener1.GetNotificationCount())
	})
}

func TestEmitter_Reconcile(t *testing.T) {
	t.Parallel()

	t.Run("emitter refersh listeners in reconcile loop", func(t *testing.T) {
		t.Parallel()
		// given first reconciler loop was done
		eventBus := bus.NewEventBusMock()
		emitter := NewEmitter(eventBus, "", nil)
		emitter.Loader.Register(&dummy.DummyLoader{IdPrefix: "dummy1"})
		emitter.Loader.Register(&dummy.DummyLoader{IdPrefix: "dummy2"})

		ctx, cancel := context.WithCancel(context.Background())

		go emitter.Reconcile(ctx)

		time.Sleep(100 * time.Millisecond)
		assert.Len(t, emitter.GetListeners(), 4)

		cancel()

		// and we'll add additional new loader
		emitter.Loader.Register(&dummy.DummyLoader{IdPrefix: "dummy1"}) // existing one
		emitter.Loader.Register(&dummy.DummyLoader{IdPrefix: "dummy3"})

		ctx, cancel = context.WithCancel(context.Background())

		// when
		go emitter.Reconcile(ctx)

		// then each reconciler (3 reconcilers) should load 2 listeners
		time.Sleep(100 * time.Millisecond)
		assert.Len(t, emitter.GetListeners(), 6)

		cancel()
	})

}

func newExampleTestEvent1() testkube.Event {
	return testkube.Event{
		Id:            "eventID1",
		Type_:         testkube.EventStartTest,
		TestExecution: testkube.NewExecutionWithID("executionID1", "test/test", "test"),
	}
}

func newExampleTestEvent2() testkube.Event {
	return testkube.Event{
		Id:            "eventID2",
		Type_:         testkube.EventStartTest,
		TestExecution: testkube.NewExecutionWithID("executionID2", "test/test", "test"),
	}
}

func TestEmitter_UpdateListeners(t *testing.T) {
	t.Parallel()

	t.Run("add, update and delete new listeners", func(t *testing.T) {
		t.Parallel()
		// given
		eventBus := bus.NewEventBusMock()
		emitter := NewEmitter(eventBus, "", nil)
		// given listener with matching selector
		listener1 := &dummy.DummyListener{Id: "l1", SelectorString: "type=listener1"}
		// and listener with second matching selector
		listener2 := &dummy.DummyListener{Id: "l2", SelectorString: "type=listener2"}
		// and listener with third matching selector
		listener3 := &dummy.DummyListener{Id: "l1", SelectorString: "type=listener3"}
		// and listener with different kind
		listener4 := &FakeListener{name: "l4"}
		// and listener with different kind
		listener5 := &FakeListener{name: "l5"}

		// when listeners are added
		emitter.UpdateListeners(common.Listeners{listener1, listener2})

		// then should have 2 listeners
		assert.Len(t, emitter.Listeners, 2)

		// when listeners are deleted
		emitter.UpdateListeners(common.Listeners{listener1})
		assert.Equal(t, "type=listener1", emitter.Listeners[0].Selector())

		// then should have 1 listener
		assert.Len(t, emitter.Listeners, 1)

		// when listeners are updated
		emitter.UpdateListeners(common.Listeners{listener3})

		// then should have 1 listener
		assert.Len(t, emitter.Listeners, 1)
		assert.Equal(t, "type=listener3", emitter.Listeners[0].Selector())

		// when listeners are added
		emitter.UpdateListeners(common.Listeners{listener3, listener2})

		// then should have 2 listeners
		assert.Len(t, emitter.Listeners, 2)

		// when listeners are added
		emitter.UpdateListeners(common.Listeners{listener4})

		// then should have 1 listeners
		assert.Len(t, emitter.Listeners, 1)

		// when listeners are added
		emitter.UpdateListeners(common.Listeners{listener4, listener5})

		// then should have 4 listeners
		assert.Len(t, emitter.Listeners, 2)
	})

}

var _ common.Listener = (*FakeListener)(nil)

type FakeListener struct {
	name string
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

func (l *FakeListener) Metadata() map[string]string {
	return map[string]string{}
}
