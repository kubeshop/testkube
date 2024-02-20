package logs

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/logs/adapter"
	"github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/state"
)

var waitTime = time.Second

func TestLogs_EventsFlow(t *testing.T) {
	t.Parallel()

	t.Run("should remove all adapters when stop event handled", func(t *testing.T) {
		// given context with 1s deadline
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
		defer cancel()

		// and NATS test server with connection
		ns, nc := bus.TestServerWithConnection()
		defer ns.Shutdown()

		id := "stop-test"

		// and jetstream configured
		js, err := jetstream.New(nc)
		assert.NoError(t, err)

		// and KV store
		kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: "state-test"})
		assert.NoError(t, err)
		assert.NotNil(t, kv)

		// and logs state manager
		state := state.NewState(kv)

		logsStream, err := client.NewNatsLogStream(nc)
		assert.NoError(t, err)

		// and initialized log service
		log := NewLogsService(nc, js, state, logsStream).
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
		stream, err := client.NewNatsLogStream(nc)
		assert.NoError(t, err)

		// and initialized log stream for given ID
		meta, err := stream.Init(ctx, id)
		assert.NotEmpty(t, meta.Name)
		assert.NoError(t, err)

		// when start event triggered
		_, err = stream.Start(ctx, id)
		assert.NoError(t, err)

		// and when data pushed to the log stream
		stream.Push(ctx, id, events.NewLog("hello 1"))
		stream.Push(ctx, id, events.NewLog("hello 2"))

		// and stop event triggered
		_, err = stream.Stop(ctx, id)
		assert.NoError(t, err)

		// cooldown stop time
		time.Sleep(waitTime)

		// then all adapters should be gracefully stopped
		assert.Equal(t, 0, log.GetConsumersStats(ctx).Count)
	})

	t.Run("should start and stop on test event", func(t *testing.T) {
		// given context with 1s deadline
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
		defer cancel()

		// and NATS test server with connection
		ns, nc := bus.TestServerWithConnection()
		defer ns.Shutdown()

		id := "id1"

		// and jetstream configured
		js, err := jetstream.New(nc)
		assert.NoError(t, err)

		// and KV store
		kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: "start-stop-on-test"})
		assert.NoError(t, err)
		assert.NotNil(t, kv)

		// and logs state manager
		state := state.NewState(kv)

		logsStream, err := client.NewNatsLogStream(nc)
		assert.NoError(t, err)

		// and initialized log service
		log := NewLogsService(nc, js, state, logsStream).
			WithRandomPort()

		// given example adapter
		a := NewMockAdapter()

		messagesCount := 10000

		// with 4 adapters (the same adapter is added 4 times so it'll receive 4 times more messages)
		log.AddAdapter(a)

		// and log service running
		go func() {
			log.Run(ctx)
		}()

		// and test event emitter
		ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
		assert.NoError(t, err)
		eventBus := bus.NewNATSBus(ec)
		emitter := event.NewEmitter(eventBus, "test-cluster", map[string]string{})

		// and stream client
		stream, err := client.NewNatsLogStream(nc)
		assert.NoError(t, err)

		// and initialized log stream for given ID
		meta, err := stream.Init(ctx, id)
		assert.NotEmpty(t, meta.Name)
		assert.NoError(t, err)

		// and ready to get messages
		<-log.Ready

		// when start event triggered
		emitter.Notify(testkube.NewEventStartTest(&testkube.Execution{Id: "id1"}))

		for i := 0; i < messagesCount; i++ {
			// and when data pushed to the log stream
			err = stream.Push(ctx, id, events.NewLog("hello"))
			assert.NoError(t, err)
		}

		// and wait for message to be propagated
		emitter.Notify(testkube.NewEventEndTestFailed(&testkube.Execution{Id: "id1"}))

		time.Sleep(waitTime)

		assertMessagesCount(t, a, messagesCount)

	})

	t.Run("should react on new message and pass data to adapter", func(t *testing.T) {
		// given context with 1s deadline
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
		defer cancel()

		// and NATS test server with connection
		ns, nc := bus.TestServerWithConnection()
		defer ns.Shutdown()

		id := "messages-test"

		// and jetstream configured
		js, err := jetstream.New(nc)
		assert.NoError(t, err)

		// and KV store
		kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: "state-test"})
		assert.NoError(t, err)
		assert.NotNil(t, kv)

		// and logs state manager
		state := state.NewState(kv)

		logsStream, err := client.NewNatsLogStream(nc)
		assert.NoError(t, err)

		// and initialized log service
		log := NewLogsService(nc, js, state, logsStream).
			WithRandomPort()

		// given example adapter
		a1 := NewMockAdapter()
		a2 := NewMockAdapter()
		a3 := NewMockAdapter()
		a4 := NewMockAdapter()

		messagesCount := 1000

		// with 4 adapters (the same adapter is added 4 times so it'll receive 4 times more messages)
		log.AddAdapter(a1)
		log.AddAdapter(a2)
		log.AddAdapter(a3)
		log.AddAdapter(a4)

		// and log service running
		go func() {
			log.Run(ctx)
		}()

		// and ready to get messages
		<-log.Ready

		// and stream client
		stream, err := client.NewNatsLogStream(nc)
		assert.NoError(t, err)

		// and initialized log stream for given ID
		meta, err := stream.Init(ctx, id)
		assert.NotEmpty(t, meta.Name)
		assert.NoError(t, err)

		// when start event triggered
		_, err = stream.Start(ctx, id)
		assert.NoError(t, err)

		for i := 0; i < messagesCount; i++ {
			// and when data pushed to the log stream
			err = stream.Push(ctx, id, events.NewLog("hello"))
			assert.NoError(t, err)
		}

		// and wait for message to be propagated
		_, err = stream.Stop(ctx, id)
		assert.NoError(t, err)

		// cool down
		time.Sleep(waitTime)

		// then each adapter should receive messages
		assertMessagesCount(t, a1, messagesCount)
		assertMessagesCount(t, a2, messagesCount)
		assertMessagesCount(t, a3, messagesCount)
		assertMessagesCount(t, a4, messagesCount)

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

		id := "executionid1"

		// and KV store
		kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: "state-test"})
		assert.NoError(t, err)
		assert.NotNil(t, kv)

		// and logs state manager
		state := state.NewState(kv)

		logsStream, err := client.NewNatsLogStream(nc)
		assert.NoError(t, err)

		// and initialized log service
		log := NewLogsService(nc, js, state, logsStream).
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
		stream, err := client.NewNatsLogStream(nc)
		assert.NoError(t, err)

		// and initialized log stream for given ID
		meta, err := stream.Init(ctx, id)
		assert.NotEmpty(t, meta.Name)
		assert.NoError(t, err)

		// when start event triggered
		_, err = stream.Start(ctx, id)
		assert.NoError(t, err)

		// then we should have 2 consumers
		stats := log.GetConsumersStats(ctx)
		assert.Equal(t, 2, stats.Count)

		stream.Push(ctx, id, events.NewLog("hello 1"))
		stream.Push(ctx, id, events.NewLog("hello 1"))
		stream.Push(ctx, id, events.NewLog("hello 1"))

		// when stop event triggered
		r, err := stream.Stop(ctx, id)
		assert.NoError(t, err)
		assert.False(t, r.Error)
		assert.Equal(t, "stop-queued", string(r.Message))

		// there will be wait for mess
		time.Sleep(waitTime)

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
		Messages: []events.Log{},
		name:     n,
	}
}

type MockAdapter struct {
	lock     sync.Mutex
	Messages []events.Log
	name     string
}

func (s *MockAdapter) Init(ctx context.Context, id string) error {
	return nil
}

func (s *MockAdapter) Notify(ctx context.Context, id string, e events.Log) error {
	// don't count finished logs
	if events.IsFinished(&e) {
		return nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	e.Metadata = map[string]string{"id": id}
	s.Messages = append(s.Messages, e)
	return nil
}

func (s *MockAdapter) Stop(ctx context.Context, id string) error {
	fmt.Printf("stopping %s \n", id)
	return nil
}

func (s *MockAdapter) Name() string {
	return s.name
}

func assertMessagesCount(t *testing.T, a *MockAdapter, expectedCount int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	for {

		select {
		case <-ctx.Done():
			t.Errorf("timeout waiting for messages count %d (expected:%d)", len(a.Messages), expectedCount)
			t.Fail()
			return
		case <-ticker.C:
			if len(a.Messages) == expectedCount {
				return
			}
		}
	}
}
