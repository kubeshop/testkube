package event

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

type DummyListener struct {
	Id                string
	NotificationCount int
	SelectorString    string
}

func (l *DummyListener) Notify(event testkube.Event) testkube.EventResult {
	fmt.Printf("%+v\n", "NOTIFYING")

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

func TestEmitter_Register(t *testing.T) {
	t.Run("adds new listener", func(t *testing.T) {
		// given
		emitter := NewEmitter()

		// when
		emitter.Register(&DummyListener{})

		// then
		assert.Equal(t, 1, len(emitter.Listeners))
	})
}

func TestEmitter_Notify(t *testing.T) {

	t.Run("notifies listeners", func(t *testing.T) {
		// given 2 listeners subscribed to the same queue
		emitter := NewEmitter()

		listener1 := &DummyListener{}
		listener2 := &DummyListener{}

		emitter.Register(listener1)
		emitter.Listen()

		// when event sent
		emitter.Notify(newExampleTestEvent1())

		// wait for Results
		<-emitter.Results

		// then only one listener should be notified
		assert.Equal(t, 1, listener2.NotificationCount+listener1.NotificationCount)
	})
}

func TestEmitter_Selectors(t *testing.T) {

	t.Run("listener handles only given events based on selectors", func(t *testing.T) {
		// given
		emitter := NewEmitter()

		// listener with matching selector
		listener1 := &DummyListener{}
		listener1.SelectorString = "type=OnlyMe"
		listener1.Id = "1"

		// listener with non matching selector
		listener2 := &DummyListener{}
		listener2.SelectorString = "type=NotMe"
		listener2.Id = "2"

		// emitter with registered listeners
		emitter.Register(listener1)
		emitter.Register(listener2)

		// listening emitter
		emitter.Listen()

		// events
		event1 := newExampleTestEvent1()
		event1.Execution.Labels = map[string]string{"type": "OnlyMe"}
		event2 := newExampleTestEvent2()

		// when
		emitter.Notify(event1)
		emitter.Notify(event2)

		// then

		// make sure all workers are done for one listener, wait for them to complete
		result := <-emitter.Results

		assert.Equal(t, 1, listener1.NotificationCount)
		assert.Equal(t, 0, listener2.NotificationCount)
		assert.Equal(t, "1", result.Id)
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
		Id:        "1",
		Type_:     testkube.EventStartTest,
		Execution: testkube.NewQueuedExecution(),
	}
}

func newExampleTestEvent2() testkube.Event {
	return testkube.Event{
		Id:        "2",
		Type_:     testkube.EventStartTest,
		Execution: testkube.NewQueuedExecution(),
	}
}
