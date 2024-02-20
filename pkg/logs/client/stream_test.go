package client

import (
	"context"
	"fmt"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/logs/events"
)

func TestStream_StartStop(t *testing.T) {
	t.Run("start and stop events are triggered", func(t *testing.T) {
		// given nats server with jetstream
		ns, nc := bus.TestServerWithConnection()
		defer ns.Shutdown()

		id := "111"

		ctx := context.Background()

		// and log stream
		client, err := NewNatsLogStream(nc)
		assert.NoError(t, err)

		// initialized
		meta, err := client.Init(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, StreamPrefix+id, meta.Name)

		// when data are passed
		err = client.PushBytes(ctx, id, []byte(`{"resourceId":"hello 1"}`))
		assert.NoError(t, err)

		var startReceived, stopReceived bool

		_, err = nc.Subscribe(StartSubject, func(m *nats.Msg) {
			m.Respond([]byte("ok"))
			startReceived = true
		})
		assert.NoError(t, err)
		_, err = nc.Subscribe(StopSubject, func(m *nats.Msg) {
			m.Respond([]byte("ok"))
			stopReceived = true
		})

		assert.NoError(t, err)

		// and stream started
		d, err := client.Start(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, "ok", string(d.Message))

		// and stream stopped
		d, err = client.Stop(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, "ok", string(d.Message))

		// then start/stop subjects should be notified
		assert.True(t, startReceived)
		assert.True(t, stopReceived)
	})

	t.Run("channel is closed when log is finished", func(t *testing.T) {
		// given nats server with jetstream
		ns, nc := bus.TestServerWithConnection()
		defer ns.Shutdown()

		id := "222"

		ctx := context.Background()

		// and log stream
		client, err := NewNatsLogStream(nc)
		assert.NoError(t, err)

		// initialized
		meta, err := client.Init(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, StreamPrefix+id, meta.Name)

		// when messages are sent
		err = client.Push(ctx, id, events.NewLog("log line 1"))
		assert.NoError(t, err)
		err = client.Push(ctx, id, events.NewLog("log line 2"))
		assert.NoError(t, err)
		err = client.Push(ctx, id, events.NewLog("log line 3"))
		assert.NoError(t, err)
		// and stream is set as finished
		err = client.Finish(ctx, id)
		assert.NoError(t, err)

		// and replay of messages is done
		ch, err := client.Get(ctx, id)
		assert.NoError(t, err)

		messagesCount := 0

		for l := range ch {
			fmt.Printf("%+v\n", l)
			messagesCount++
			if events.IsFinished(&l.Log) {
				break
			}
		}

		// then
		assert.Equal(t, 3, messagesCount)
	})
}

func TestStream_Name(t *testing.T) {
	client, err := NewNatsLogStream(nil)
	assert.NoError(t, err)

	t.Run("passed one string param", func(t *testing.T) {
		name := client.Name("111")
		assert.Equal(t, StreamPrefix+"111", name)
	})

	t.Run("passed no string params generates random name", func(t *testing.T) {
		name := client.Name()
		assert.Len(t, name, len(StreamPrefix)+10)
	})

	t.Run("passed more string params ignore rest", func(t *testing.T) {
		name := client.Name("111", "222", "333")
		assert.Equal(t, StreamPrefix+"111", name)
	})

}
