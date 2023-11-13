package logs

import (
	"context"
	"testing"
	"time"

	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
)

func TestStream_StartStop(t *testing.T) {
	ns, nc := bus.TestServerWithConnection()
	defer ns.Shutdown()

	js, err := jetstream.New(nc)
	assert.NoError(t, err)
	ctx := context.Background()
	stream := NewNATSStream(nc, js, "111")
	stream.Init(ctx)
	stream.Push(ctx, []byte(`{"content":"hello 1"}`))

	var startReceived, stopReceived bool

	_, err = nc.Subscribe(StartSubject, func(m *nats.Msg) {
		startReceived = true
	})
	assert.NoError(t, err)
	_, err = nc.Subscribe(StopSubject, func(m *nats.Msg) {
		stopReceived = true
	})
	assert.NoError(t, err)

	err = stream.Start(ctx)
	assert.NoError(t, err)

	err = stream.Stop(ctx)
	assert.NoError(t, err)

	time.Sleep(time.Second * 1)

	assert.True(t, startReceived)
	assert.True(t, stopReceived)
}
