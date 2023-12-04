package logs

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/logs/adapter"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/state"
)

func TestLogs_EventsFlow(t *testing.T) {
	t.Parallel()

	t.Run("should remove all adapters when stop event handled", func(t *testing.T) {
		// given context with 1s deadline
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
		defer cancel()

		// and NATS test server with connection
		ns, nc := bus.TestServerWithConnection()
		defer ns.Shutdown()

		// and jetstream configured
		js, err := jetstream.New(nc)
		assert.NoError(t, err)

		// and KV store
		kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: "state-test"})
		assert.NoError(t, err)
		assert.NotNil(t, kv)

		// and logs state manager
		state := state.NewState(kv)

		// and initialized log service
		log := NewLogsService(nc, js, state).
			WithRandomPort()

		// given example adapters
		a := NewMockAdapter("aaa")
		b := NewMockAdapter("bbb")

		// with 4 adapters (the same adapter is added 4 times so it'll receive 4 times more messages)
		log.AddAdapter(a)
		log.AddAdapter(b)

		// and log service running
		go func() {
			log.Run(ctx)
		}()

		// and ready to get messages
		<-log.Ready

		// and logs stream client
		stream, err := NewLogsStream(nc, "stop-test")
		assert.NoError(t, err)

		// and initialized log stream for given ID
		meta, err := stream.Init(ctx)
		assert.NotEmpty(t, meta.Name)
		assert.NoError(t, err)

		// when start event triggered
		_, err = stream.Start(ctx)
		assert.NoError(t, err)

		// and when data pushed to the log stream
		stream.Push(ctx, events.NewLogChunk(time.Now(), []byte("hello 1")))

		// and stop event triggered
		_, err = stream.Stop(ctx)
		assert.NoError(t, err)

		// then all adapters should be gracefully stopped
		assert.Equal(t, 0, log.GetConsumersStats(ctx).Count)
	})

	t.Run("should react on new message and pass data to adapter", func(t *testing.T) {
		// given context with 1s deadline
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
		defer cancel()

		// and NATS test server with connection
		ns, nc := bus.TestServerWithConnection()
		defer ns.Shutdown()

		// and jetstream configured
		js, err := jetstream.New(nc)
		assert.NoError(t, err)

		// and KV store
		kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: "state-test"})
		assert.NoError(t, err)
		assert.NotNil(t, kv)

		// and logs state manager
		state := state.NewState(kv)

		// and initialized log service
		log := NewLogsService(nc, js, state).
			WithRandomPort()

		// given example adapter
		a := NewMockAdapter()

		messagesCount := 10000

		// with 4 adapters (the same adapter is added 4 times so it'll receive 4 times more messages)
		log.AddAdapter(a)
		log.AddAdapter(a)
		log.AddAdapter(a)
		log.AddAdapter(a)

		// and log service running
		go func() {
			log.Run(ctx)
		}()

		// and ready to get messages
		<-log.Ready

		// and stream client
		stream, err := NewLogsStream(nc, "messages-test")
		assert.NoError(t, err)

		// and initialized log stream for given ID
		meta, err := stream.Init(ctx)
		assert.NotEmpty(t, meta.Name)
		assert.NoError(t, err)

		// when start event triggered
		_, err = stream.Start(ctx)
		assert.NoError(t, err)

		for i := 0; i < messagesCount; i++ {
			// and when data pushed to the log stream
			err = stream.Push(ctx, events.NewLogChunk(time.Now(), []byte("hello")))
			assert.NoError(t, err)
		}

		// and wait for message to be propagated
		_, err = stream.Stop(ctx)
		assert.NoError(t, err)

		// then we should have 4*4 messages in adapter
		assert.Equal(t, 4*messagesCount, len(a.Messages))
	})

	t.Run("can get stats about consumers in given pod", func(t *testing.T) {
		// given context with 1s deadline
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
		defer cancel()

		// and NATS test server with connection
		ns, nc := bus.TestServerWithConnection()
		defer ns.Shutdown()

		// and jetstream configured
		js, err := jetstream.New(nc)
		assert.NoError(t, err)

		// and KV store
		kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: "state-test"})
		assert.NoError(t, err)
		assert.NotNil(t, kv)

		// and logs state manager
		state := state.NewState(kv)

		// and initialized log service
		log := NewLogsService(nc, js, state).
			WithRandomPort()

		// given example adapters
		a := NewMockAdapter("aaa")
		b := NewMockAdapter("bbb")

		// with 4 adapters (the same adapter is added 4 times so it'll receive 4 times more messages)
		log.AddAdapter(a)
		log.AddAdapter(b)

		// and log service running
		go func() {
			log.Run(ctx)
		}()

		// and ready to get messages
		<-log.Ready

		// and logs stream client
		stream, err := NewLogsStream(nc, "stop-test")
		assert.NoError(t, err)

		// and initialized log stream for given ID
		meta, err := stream.Init(ctx)
		assert.NotEmpty(t, meta.Name)
		assert.NoError(t, err)

		// when start event triggered
		_, err = stream.Start(ctx)
		assert.NoError(t, err)

		// then we should have 2 consumers
		stats := log.GetConsumersStats(ctx)
		assert.Equal(t, 2, stats.Count)

		// when stop event triggered
		_, err = stream.Stop(ctx)
		assert.NoError(t, err)

		// then all adapters should be gracefully stopped
		assert.Equal(t, 0, log.GetConsumersStats(ctx).Count)
	})

}

// Mock adapter
var _ adapter.Adapter = &MockAdapter{}

// NewMockAdapter creates new mocked adapter to check amount of messages passed to it
func NewMockAdapter(name ...string) *MockAdapter {
	n := "default"
	if len(name) > 0 {
		n = name[0]
	}

	return &MockAdapter{
		Messages: []events.LogChunk{},
		name:     n,
	}
}

type MockAdapter struct {
	lock     sync.Mutex
	Messages []events.LogChunk
	name     string
}

func (s *MockAdapter) Notify(id string, e events.LogChunk) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	e.Metadata = map[string]string{"id": id}
	s.Messages = append(s.Messages, e)
	return nil
}

func (s *MockAdapter) Stop(id string) error {
	fmt.Printf("stopping %s \n", id)
	return nil
}

func (s *MockAdapter) Name() string {
	return s.name
}
