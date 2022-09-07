package bus

import (
	"sync"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

func TestNATS(t *testing.T) {

	// given event

	event := testkube.NewEventStartTest(testkube.NewQueuedExecution())
	event.Id = "123"

	// and connection
	nc, err := nats.Connect("localhost")
	assert.NoError(t, err)
	defer nc.Close()

	// and automatic JSON encoder
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	assert.NoError(t, err)
	defer ec.Close()

	// and NATS event bus
	n := NewNATSEventBus(ec)

	// when 2 subscriptions are made
	ch1, err := n.Subscribe(event.Type(), "test1")
	assert.NoError(t, err)
	ch2, err := n.Subscribe(event.Type(), "test2")
	assert.NoError(t, err)

	var wg sync.WaitGroup

	// and event is published to event bus
	err = n.Publish(event)
	assert.NoError(t, err)

	// then 2 events are received
	wg.Add(1)
	go func() {
		ch1Event := <-ch1
		assert.Equal(t, "123", ch1Event.Id)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		ch2Event := <-ch2
		assert.Equal(t, "123", ch2Event.Id)
		wg.Done()
	}()

	// and wait for events to be received
	wg.Wait()
}
