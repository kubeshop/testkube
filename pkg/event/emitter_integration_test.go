//go:build integration

package event

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/dummy"
	"github.com/stretchr/testify/assert"
)

// tests based on real NATS event bus
var eventBus bus.Bus

func init() {
	os.Setenv("DEBUG", "true")
	// configure NATS event bus
	nc, err := bus.NewNATSConnection()
	if err != nil {
		log.DefaultLogger.Errorw("error creating NATS connection", "error", err)
	}
	eventBus = bus.NewNATSBus(nc)
}

func TestEmitter_NATS_Register(t *testing.T) {

	t.Run("Register adds new listener", func(t *testing.T) {
		// given
		emitter := NewEmitter(eventBus)
		// when
		emitter.Register(&dummy.DummyListener{Id: "l1"})

		// then
		assert.Equal(t, 1, len(emitter.Listeners))

		t.Log("T1 completed")
	})
}

func TestEmitter_NATS_Listen(t *testing.T) {
	t.Run("listener handles only given events based on selectors", func(t *testing.T) {
		// given
		emitter := NewEmitter(eventBus)
		// given listener with matching selector
		listener1 := &dummy.DummyListener{Id: "l1", SelectorString: "type=OnlyMe"}
		// and listener with non matching selector
		listener2 := &dummy.DummyListener{Id: "l2", SelectorString: "type=NotMe"}

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
		event1.Execution.Labels = map[string]string{"type": "OnlyMe"}
		event2 := newExampleTestEvent2()

		// when
		emitter.Notify(event1)
		emitter.Notify(event2)

		// then
		time.Sleep(time.Millisecond * 100)

		assert.Equal(t, 1, listener1.GetNotificationCount())
		assert.Equal(t, 0, listener2.GetNotificationCount())
		t.Log("T3 completed")
	})

}

func TestEmitter_NATS_Notify(t *testing.T) {
	t.Run("notifies listeners in queue groups", func(t *testing.T) {
		// given
		emitter := NewEmitter(eventBus)
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
		// wait for listeners to start
		time.Sleep(time.Millisecond * 50)

		// when event sent to queue group
		emitter.Notify(newExampleTestEvent1())

		time.Sleep(time.Millisecond * 100)

		// then only one listener should be notified
		assert.Equal(t, 1, listener2.GetNotificationCount()+listener1.GetNotificationCount())
	})
}

func TestEmitter_NATS_Reconcile(t *testing.T) {

	t.Run("emitter refersh listeners", func(t *testing.T) {
		// given
		emitter := NewEmitter(eventBus)
		emitter.Loader.Register(&dummy.DummyLoader{})
		emitter.Loader.Register(&dummy.DummyLoader{})

		ctx, cancel := context.WithCancel(context.Background())

		// when
		go emitter.Reconcile(ctx)

		// then
		time.Sleep(time.Millisecond)
		assert.Len(t, emitter.Listeners, 4)

		cancel()
	})

	t.Run("emitter refersh listeners in reconcile loop", func(t *testing.T) {
		// given first reconciler loop was done
		emitter := NewEmitter(eventBus)
		emitter.Loader.Register(&dummy.DummyLoader{})
		emitter.Loader.Register(&dummy.DummyLoader{})

		ctx, cancel := context.WithCancel(context.Background())

		go emitter.Reconcile(ctx)

		time.Sleep(time.Millisecond)
		assert.Len(t, emitter.Listeners, 4)

		cancel()

		// and we'll add additional reconcilers
		emitter.Loader.Register(&dummy.DummyLoader{})
		emitter.Loader.Register(&dummy.DummyLoader{})

		ctx, cancel = context.WithCancel(context.Background())

		// when
		go emitter.Reconcile(ctx)

		// then each reconciler (4 reconcilers) should load 2 listeners
		time.Sleep(time.Millisecond)
		assert.Len(t, emitter.Listeners, 8)

		cancel()
	})

}

func newExampleTestEvent1() testkube.Event {
	return testkube.Event{
		Id:        "eventID1",
		Type_:     testkube.EventStartTest,
		Execution: testkube.NewExecutionWithID("executionID1", "test/test", "test"),
	}
}

func newExampleTestEvent2() testkube.Event {
	return testkube.Event{
		Id:        "eventID2",
		Type_:     testkube.EventStartTest,
		Execution: testkube.NewExecutionWithID("executionID2", "test/test", "test"),
	}
}

func TestEmitter_NATS_Register(t *testing.T) {

	wsl := websockets.NewWebsocketLoader()
	e := NewEmitter()
	e.Loader.Register(wsl)

}
