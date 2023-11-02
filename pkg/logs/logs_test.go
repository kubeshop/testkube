package logs

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	n "github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/logs/consumer"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
)

func TestLogs(t *testing.T) {

	ns, nc := n.TestServerWithConnection()
	defer ns.Shutdown()

	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	assert.NoError(t, err)

	js, err := jetstream.New(nc)
	assert.NoError(t, err)

	t.Run("should react on new message and pass data to consumer", func(t *testing.T) {

		// given one second context
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// and example consumer
		c := NewMockConsumer()

		// and initialized log service
		log := NewLogsService(ec, js)

		// with 4 consumers (the same consumer is added 4 times so it'll receive 4 times more messages)
		log.AddConsumer(c)
		log.AddConsumer(c)
		log.AddConsumer(c)
		log.AddConsumer(c)

		// and log service running
		go func() {
			log.Run(ctx)
		}()

		// and ready to get messages
		<-log.Ready

		// when we publish start event
		err := ec.Publish(StartTopic, events.Trigger{Id: "123"})
		assert.NoError(t, err)

		// and push logs to given logs stream
		streamName := StreamName + "123"
		_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
			Name:    streamName,
			Storage: jetstream.FileStorage, // durable stream
		})
		assert.NoError(t, err)

		// and line by line we generate 4 log lines
		_, err = js.Publish(ctx, streamName, []byte(`{"content":"hello 1"}`))
		assert.NoError(t, err)
		_, err = js.Publish(ctx, streamName, []byte(`{"content":"hello 2"}`))
		assert.NoError(t, err)
		_, err = js.Publish(ctx, streamName, []byte(`{"content":"hello 3"}`))
		assert.NoError(t, err)
		_, err = js.Publish(ctx, streamName, []byte(`{"content":"hello 4"}`))
		assert.NoError(t, err)

		// and we stop propagating log messages
		err = ec.Publish(StopTopic, events.Trigger{Id: "123"})
		assert.NoError(t, err)

		// and wait for messages to be propagated
		time.Sleep(100 * time.Millisecond)

		// then we should have 4*4 messages in consumer
		assert.Equal(t, 16, len(c.Messages))
	})
}

// Mock consumer
var _ consumer.Consumer = &MockConsumer{}

// NewMockConsumer creates new mocked consumer to check amount of messages passed to it
func NewMockConsumer() *MockConsumer {
	return &MockConsumer{
		Messages: []events.LogChunk{},
	}
}

type MockConsumer struct {
	lock     sync.Mutex
	Messages []events.LogChunk
}

func (s *MockConsumer) Notify(id string, e events.LogChunk) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	e.Metadata = map[string]string{"id": id}
	s.Messages = append(s.Messages, e)
	return nil
}

func (s *MockConsumer) Stop(id string) error {
	fmt.Printf("stopping %s \n", id)
	return nil
}

func (s *MockConsumer) Name() string {
	return "mock"
}
