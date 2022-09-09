package event

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("DEBUG", "true")
}

var _ common.Listener = &DummyListener{}

type DummyListener struct {
	Id                string
	NotificationCount int32
	SelectorString    string
}

func (l *DummyListener) GetNotificationCount() int {
	cnt := atomic.LoadInt32(&l.NotificationCount)
	return int(cnt)
}

func (l *DummyListener) Notify(event testkube.Event) testkube.EventResult {
	atomic.AddInt32(&l.NotificationCount, 1)
	return testkube.EventResult{Id: event.Id}
}

func (l *DummyListener) Name() string {
	if l.Id != "" {
		return l.Id
	}
	return "dummy"
}

func (l *DummyListener) Events() []testkube.EventType {
	return testkube.AllEventTypes
}

func (l *DummyListener) Selector() string {
	return l.SelectorString
}

func (l *DummyListener) Kind() string {
	return "dummy"
}

func (l *DummyListener) Metadata() map[string]string {
	return map[string]string{"uri": "http://localhost:8080"}
}

func TestEmitter_SendXMessages(t *testing.T) {
	// given

	emitter := NewEmitter()

	// given listener with matching selector
	listener1 := &DummyListener{Id: "id-1"}

	// and emitter with registered listeners
	emitter.Register(listener1)

	// listening emitter
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	emitter.Listen(ctx)

	// events
	emitter.Notify(newExampleTestEvent1())

	// wait for messages to completed
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, listener1.GetNotificationCount())
}

func TestEmitter_Register(t *testing.T) {

	t.Run("Register adds new listener", func(t *testing.T) {
		// given
		emitter := NewEmitter()
		// when
		emitter.Register(&DummyListener{Id: "l1"})

		// then
		assert.Equal(t, 1, len(emitter.Listeners))

		t.Log("T1 completed")
	})
}

func TestEmitter_Listen(t *testing.T) {
	t.Run("listener handles only given events based on selectors", func(t *testing.T) {
		// given
		emitter := NewEmitter()
		// given listener with matching selector
		listener1 := &DummyListener{Id: "l1", SelectorString: "type=OnlyMe"}
		// and listener with non matching selector
		listener2 := &DummyListener{Id: "l2", SelectorString: "type=NotMe"}

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
		time.Sleep(time.Millisecond * 100)

		assert.Equal(t, 1, listener1.GetNotificationCount())
		assert.Equal(t, 0, listener2.GetNotificationCount())
		t.Log("T3 completed")
	})

}

func TestEmitter_Notify(t *testing.T) {
	t.Run("notifies listeners in queue groups", func(t *testing.T) {
		// given
		emitter := NewEmitter()
		// and 2 listeners subscribed to the same queue
		// * first on pod1
		listener1 := &DummyListener{Id: "l3"}
		// * second on pod2
		listener2 := &DummyListener{Id: "l3"}

		emitter.Register(listener1)
		emitter.Register(listener2)

		// and listening emitter
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		emitter.Listen(ctx)

		// when event sent to queue group
		emitter.Notify(newExampleTestEvent1())

		time.Sleep(time.Millisecond * 100)

		// then only one listener should be notified
		assert.Equal(t, 1, listener2.GetNotificationCount()+listener1.GetNotificationCount())
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
