package event

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("DEBUG", "true")
}

var _ common.Listener = &DummyListener{}

type DummyListener struct {
	Id                string
	NotificationCount int
	SelectorString    string
}

func (l *DummyListener) Notify(event testkube.Event) testkube.EventResult {
	l.NotificationCount++
	return testkube.EventResult{Id: event.Id}
}

func (l DummyListener) Name() string {
	if l.Id != "" {
		return l.Id
	}
	return "dummy"
}

func (l DummyListener) Event() testkube.EventType {
	return testkube.START_TEST_EventType
}

func (l DummyListener) Selector() string {
	return l.SelectorString
}

func (l DummyListener) Kind() string {
	return "dummy"
}

func (l DummyListener) Metadata() map[string]string {
	return map[string]string{"uri": "http://localhost:8080"}
}

func TestEmitter_SendXMessages(t *testing.T) {
	// given
	emitter := NewEmitter()
	// given listener with matching selector
	i := rand.Intn(10000000)
	listener1 := &DummyListener{Id: fmt.Sprintf("id-%d", i)}

	// and emitter with registered listeners
	emitter.Register(listener1)

	// listening emitter
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	emitter.Listen(ctx)

	// events
	evt := newExampleTestEvent1()

	eventsCount := 1
	// when sending X events
	for i := 0; i < eventsCount; i++ {
		log.DefaultLogger.Infof("sending event %d", i)
		evt.Id = fmt.Sprintf("EventID%d", i)
		emitter.Notify(evt)
	}

	// then there should be X notifications
	for i := 0; i < eventsCount; i++ {
		fmt.Printf("waiting for event %+v\n", i)
		<-emitter.Results
		fmt.Printf("got result %+v\n", i)
	}

	assert.Equal(t, eventsCount, listener1.NotificationCount)

	t.Log("T4 completed")

}

func TestEmitter_Listen(t *testing.T) {

	t.Run("Register adds new listener", func(t *testing.T) {
		t.Skip("")
		// given
		emitter := NewEmitter()
		// when
		emitter.Register(&DummyListener{Id: "l1"})

		// then
		assert.Equal(t, 1, len(emitter.Listeners))

		t.Log("T1 completed")
	})

	t.Run("listener handles only given events based on selectors", func(t *testing.T) {
		t.Skip("")
		// given
		emitter := NewEmitter()
		// given listener with matching selector
		listener1 := &DummyListener{Id: "l4", SelectorString: "type=OnlyMe"}
		// and listener with non matching selector
		listener2 := &DummyListener{Id: "l5", SelectorString: "type=NotMe"}

		// and emitter with registered listeners
		emitter.Register(listener1)
		emitter.Register(listener2)

		// listening emitter
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		emitter.Listen(ctx)

		// events
		event1 := newExampleTestEvent1()
		event1.Execution.Labels = map[string]string{"type": "OnlyMe"}
		event2 := newExampleTestEvent2()

		// when
		emitter.Notify(event1)
		emitter.Notify(event2)

		// then

		// make sure all workers are done for one listener, wait for them to complete
		go func() {
			for {
				fmt.Printf("RESULTS CHAN LEN %+v\n", len(emitter.Results))
				time.Sleep(100 * time.Millisecond)
			}
		}()

		result := <-emitter.Results
		fmt.Printf("RESULT: %+v\n", result)
		assert.Equal(t, "eventID1", result.Id)

		time.Sleep(time.Second)

		fmt.Printf("LISTENER1: %+v\n", listener1)
		fmt.Printf("LISTENER2: %+v\n", listener2)

		assert.Equal(t, 1, listener1.NotificationCount)
		assert.Equal(t, 0, listener2.NotificationCount)
		t.Log("T3 completed")
	})

	t.Run("notifies listeners", func(t *testing.T) {
		t.Skip("")
		// given
		emitter := NewEmitter()
		// given 2 listeners subscribed to the same queue
		listener1 := &DummyListener{Id: "l2"}
		listener2 := &DummyListener{Id: "l3"}

		emitter.Register(listener1)
		emitter.Register(listener2)

		// listening emitter
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		emitter.Listen(ctx)

		// when event sent
		emitter.Notify(newExampleTestEvent1())

		// wait for Results
		result := <-emitter.Results
		fmt.Printf("RESULT: %+v\n", result)

		// then only one listener should be notified
		assert.Equal(t, 1, listener2.NotificationCount+listener1.NotificationCount)
		t.Log("T2 completed")

	})

	t.Run("listener handles only given events based on selectors", func(t *testing.T) {
		t.Skip("")
		// given
		emitter := NewEmitter()
		// given listener with matching selector
		listener1 := &DummyListener{Id: "l4", SelectorString: "type=OnlyMe"}
		// and listener with non matching selector
		listener2 := &DummyListener{Id: "l5", SelectorString: "type=NotMe"}

		// and emitter with registered listeners
		emitter.Register(listener1)
		emitter.Register(listener2)

		// listening emitter
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		emitter.Listen(ctx)

		// events
		event1 := newExampleTestEvent1()
		event1.Execution.Labels = map[string]string{"type": "OnlyMe"}
		event2 := newExampleTestEvent2()

		// when
		emitter.Notify(event1)
		emitter.Notify(event2)

		// then

		// make sure all workers are done for one listener, wait for them to complete
		go func() {
			for {
				fmt.Printf("RESULTS CHAN LEN %+v\n", len(emitter.Results))
				time.Sleep(100 * time.Millisecond)
			}
		}()

		result := <-emitter.Results
		fmt.Printf("RESULT: %+v\n", result)
		assert.Equal(t, "eventID1", result.Id)

		time.Sleep(time.Second)

		fmt.Printf("LISTENER1: %+v\n", listener1)
		fmt.Printf("LISTENER2: %+v\n", listener2)

		assert.Equal(t, 1, listener1.NotificationCount)
		assert.Equal(t, 0, listener2.NotificationCount)
		t.Log("T3 completed")
	})
}

func TestEmitter_Reconcile(t *testing.T) {

	t.Run("emitter refersh listeners", func(t *testing.T) {
		// given
		emitter := NewEmitter()
		emitter.Loader.Register(&DummyLoader{})
		emitter.Loader.Register(&DummyLoader{})

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
		emitter := NewEmitter()
		emitter.Loader.Register(&DummyLoader{})
		emitter.Loader.Register(&DummyLoader{})

		ctx, cancel := context.WithCancel(context.Background())

		go emitter.Reconcile(ctx)

		time.Sleep(time.Millisecond)
		assert.Len(t, emitter.Listeners, 4)

		cancel()

		// and we'll add additional reconcilers
		emitter.Loader.Register(&DummyLoader{})
		emitter.Loader.Register(&DummyLoader{})

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
