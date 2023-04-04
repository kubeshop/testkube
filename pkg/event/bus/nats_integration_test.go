package bus

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestMultipleMessages_Integration(t *testing.T) {
	test.IntegrationTest(t)

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
		assert.NoError(t, json.Unmarshal(msg.Data, &event))
		atomic.AddInt32(&i, 1)
		wg.Done()
	})
	nc.QueueSubscribe("test1", "q1", func(msg *nats.Msg) {
		var event testkube.Event
		assert.NoError(t, json.Unmarshal(msg.Data, &event))
		atomic.AddInt32(&i, 1)
		wg.Done()
	})

	// second subscription with another queue group
	nc.QueueSubscribe("test1", "q2", func(msg *nats.Msg) {
		var event testkube.Event
		assert.NoError(t, json.Unmarshal(msg.Data, &event))
		atomic.AddInt32(&i, 1)
		wg.Done()
	})

	// when events are published
	for j := 0; j < eventCount; j++ {
		err := nc.Publish("test1", []byte(fmt.Sprintf(`{"id":"%d","type":"test"}`, j)))
		if err != nil {
			t.Errorf("got publish error %v", err)
		}
	}

	wg.Wait()

	// then all events are received
	// first 2 subscriptions with one queue group should have both `eventCount` messages
	// second subscription should have also `eventCount` messages
	assert.Equal(t, int32(2*eventCount), i)

}

func TestNATS_Integration(t *testing.T) {
	test.IntegrationTest(t)

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
	n := NewNATSBus(ec)

	var wg sync.WaitGroup
	wg.Add(2)
	// when 2 subscriptions are made
	err = n.Subscribe("test1", func(evt testkube.Event) error {
		assert.Equal(t, "123", evt.Id)
		wg.Done()
		return nil
	})
	assert.NoError(t, err)
	err = n.Subscribe("test2", func(evt testkube.Event) error {
		assert.Equal(t, "123", evt.Id)
		wg.Done()
		return nil
	})
	assert.NoError(t, err)

	// and event is published to event bus
	err = n.Publish(event)
	assert.NoError(t, err)

	wg.Wait()
}
