package bus

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

func TestMultipleMessages(t *testing.T) {
	// given NATS connection
	nc, err := nats.Connect("localhost")
	assert.NoError(t, err)
	defer nc.Close()

	var i int32
	var wg sync.WaitGroup

	eventCount := 1000

	// and 2 subscriptions

	wg.Add(2 * eventCount)

	// first 2 subscriptions with one queue group
	nc.QueueSubscribe("test1", "q1", func(msg *nats.Msg) {

		var event testkube.Event
		json.Unmarshal(msg.Data, &event)
		atomic.AddInt32(&i, 1)
		wg.Done()
	})
	nc.QueueSubscribe("test1", "q1", func(msg *nats.Msg) {
		var event testkube.Event
		json.Unmarshal(msg.Data, &event)
		atomic.AddInt32(&i, 1)
		wg.Done()
	})

	// second subscription with another queue group
	nc.QueueSubscribe("test1", "q2", func(msg *nats.Msg) {
		var event testkube.Event
		json.Unmarshal(msg.Data, &event)
		atomic.AddInt32(&i, 1)
		wg.Done()
	})

	// when events are published
	for j := 0; j < eventCount; j++ {
		err := nc.Publish("test1", []byte(fmt.Sprintf(`{"id":"%d","type":"test"}`, j)))
		if err != nil {
			fmt.Printf("ERR: %+v\n", err)

		}
	}

	wg.Wait()

	// then all events are received
	// first 2 subscriptions with one queue group should have both `eventCount` messages
	// second subscription should have also `eventCount` messages
	assert.Equal(t, int32(2*eventCount), i)

}

func TestNATS(t *testing.T) {

	// given event

	event := testkube.NewEventStartTest(testkube.NewQueuedExecution())
	event.Id = "123"

	// and connection
	nc, err := nats.Connect("localhost")
	assert.NoError(t, err)
	defer nc.Close()

	// and automatic JSON encoder
	ec, err := nats.NewEncodedConn(nc, nats.DEFAULT_ENCODER)
	assert.NoError(t, err)
	defer ec.Close()

	// and NATS event bus
	n := NewNATSEventBus(nc)

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
