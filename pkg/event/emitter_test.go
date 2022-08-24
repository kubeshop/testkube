package event

import (
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/stretchr/testify/assert"
)

type DummyListener struct {
	NotificationCount int
}

func (l *DummyListener) Notify(event testkube.TestkubeEvent) testkube.TestkubeEventResult {
	l.NotificationCount++
	return testkube.TestkubeEventResult{Id: "1"}
}

func (l DummyListener) Kind() ListenerKind {
	return ListenerKind("dummy")
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
		emitter.Notify(testkube.TestkubeEvent{})

		// make sure all workers are done for two listeners, wait for them to complete
		<-emitter.Results
		result := <-emitter.Results

		// then
		assert.Equal(t, 1, listener1.NotificationCount)
		assert.Equal(t, 1, listener2.NotificationCount)
		assert.Equal(t, "1", result.Id)
	})
}
