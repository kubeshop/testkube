package event

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

type DummyListener struct {
	NotificationCount int
	SelectorString    string
}

func (l *DummyListener) Notify(event testkube.TestkubeEvent) testkube.TestkubeEventResult {
	l.NotificationCount++
	return testkube.TestkubeEventResult{Id: event.Id}
}

func (l DummyListener) Events() []testkube.TestkubeEventType {
	return []testkube.TestkubeEventType{
		testkube.START_TEST_TestkubeEventType,
		testkube.END_TEST_TestkubeEventType,
	}
}

func (l DummyListener) Selector() string {
	return l.SelectorString
}

func (l DummyListener) Kind() string {
	return "dummy"
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
		// given
		emitter := NewEmitter()

		listener1 := &DummyListener{}
		listener2 := &DummyListener{}

		emitter.Register(listener1)
		emitter.Register(listener2)

		emitter.RunWorkers()

		// when
		emitter.Notify(newExampleTestEvent1())

		// make sure all workers are done for two listeners, wait for them to complete
		<-emitter.Results
		result := <-emitter.Results

		// then
		assert.Equal(t, 1, listener1.NotificationCount)
		assert.Equal(t, 1, listener2.NotificationCount)
		assert.Equal(t, "1", result.Id)
	})

	t.Run("listener handles only given events based on selectors", func(t *testing.T) {
		// given
		emitter := NewEmitter()

		listener1 := &DummyListener{}
		listener1.SelectorString = "type=OnlyMe"
		listener2 := &DummyListener{}
		listener2.SelectorString = "type=NotMe"

		emitter.Register(listener1)
		emitter.Register(listener2)

		emitter.RunWorkers()

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

func newExampleTestEvent1() testkube.TestkubeEvent {
	return testkube.TestkubeEvent{
		Id:        "1",
		Type_:     testkube.TestkubeEventStartTest,
		Execution: testkube.NewQueuedExecution(),
	}
}

func newExampleTestEvent2() testkube.TestkubeEvent {
	return testkube.TestkubeEvent{
		Id:        "2",
		Type_:     testkube.TestkubeEventStartTest,
		Execution: testkube.NewQueuedExecution(),
	}
}
