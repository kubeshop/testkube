package event

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/event/kind/dummy"
)

// tests based on real NATS event bus

func GetTestNATSEmitter() *Emitter {
	os.Setenv("DEBUG", "true")
	// configure NATS event bus
	nc, err := bus.NewNATSEncodedConnection(bus.ConnectionConfig{
		NatsURI: "http://localhost:4222",
	})

	if err != nil {
		panic(err)
	}
	return NewEmitter(bus.NewNATSBus(nc), "", nil)
}

func TestEmitter_NATS_Register_Integration(t *testing.T) {
	test.IntegrationTest(t)

	t.Run("Register adds new listener", func(t *testing.T) {
		// given
		emitter := GetTestNATSEmitter()
		// when
		emitter.Register(&dummy.DummyListener{Id: "l1"})

		// then
		assert.Equal(t, 1, len(emitter.Listeners))

		t.Log("T1 completed")
	})
}

func TestEmitter_NATS_Listen_Integration(t *testing.T) {
	test.IntegrationTest(t)

	t.Run("listener handles only given events based on selectors", func(t *testing.T) {
		// given
		emitter := GetTestNATSEmitter()
		// given listener with matching selector
		listener1 := &dummy.DummyListener{Id: "l1", SelectorString: "type=OnlyMe"}
		// and listener with non-matching selector
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
		event1 := newExampleTestEvent3()
		event1.TestExecution.Labels = map[string]string{"type": "OnlyMe"}
		event2 := newExampleTestEvent4()

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

func TestEmitter_NATS_Notify_Integration(t *testing.T) {
	test.IntegrationTest(t)

	t.Run("notifies listeners in queue groups", func(t *testing.T) {
		// given
		emitter := GetTestNATSEmitter()
		// and 2 listeners subscribed to the same queue
		// * first on pod1
		listener1 := &dummy.DummyListener{Id: "l3", NotificationCount: 0}
		// * second on pod2
		listener2 := &dummy.DummyListener{Id: "l3", NotificationCount: 0}

		emitter.Register(listener1)
		emitter.Register(listener2)

		// and listening emitter
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		emitter.Listen(ctx)
		// wait for listeners to start
		time.Sleep(time.Millisecond * 50)

		// when event sent to queue group
		emitter.Notify(newExampleTestEvent3())

		time.Sleep(time.Millisecond * 100)

		// then listeners should be notified at least once
		assert.LessOrEqual(t, 1, listener2.GetNotificationCount()+listener1.GetNotificationCount())
	})
}

func TestEmitter_NATS_Reconcile_Integration(t *testing.T) {
	test.IntegrationTest(t)

	t.Run("emitter refersh listeners in reconcile loop", func(t *testing.T) {
		// given
		emitter := GetTestNATSEmitter()
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
		event1 := newExampleTestEvent3()
		event1.TestExecution.Labels = map[string]string{"type": "listener1"}
		event2 := newExampleTestEvent4()
		event2.TestExecution.Labels = map[string]string{"type": "listener2"}

		// when
		emitter.Notify(event1)
		emitter.Notify(event2)

		time.Sleep(time.Millisecond * 50)
		// then

		assert.Equal(t, 1, listener1.GetNotificationCount())
		assert.Equal(t, 1, listener2.GetNotificationCount())
	})

}

func newExampleTestEvent3() testkube.Event {
	return testkube.Event{
		Id:            "eventID3",
		Type_:         testkube.EventStartTest,
		TestExecution: testkube.NewExecutionWithID("executionID3", "test/test", "test"),
	}
}

func newExampleTestEvent4() testkube.Event {
	return testkube.Event{
		Id:            "eventID4",
		Type_:         testkube.EventStartTest,
		TestExecution: testkube.NewExecutionWithID("executionID4", "test/test", "test"),
	}
}
