package logs

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/event/bus"
)

func TestStream_StartStop(t *testing.T) {
	ns, nc := bus.TestServerWithConnection()
	defer ns.Shutdown()

	ctx := context.Background()
	client, err := NewLogsStream(nc, "111")
	assert.NoError(t, err)
	client.Init(ctx)
	client.PushBytes(ctx, []byte(`{"content":"hello 1"}`))

	var startReceived, stopReceived bool

	_, err = nc.Subscribe(StartSubject, func(m *nats.Msg) {
		startReceived = true
	})
	assert.NoError(t, err)
	_, err = nc.Subscribe(StopSubject, func(m *nats.Msg) {
		stopReceived = true
	})
	assert.NoError(t, err)

	err = client.Start(ctx)
	assert.NoError(t, err)

	err = client.Stop(ctx)
	assert.NoError(t, err)

	time.Sleep(time.Second * 1)

	assert.True(t, startReceived)
	assert.True(t, stopReceived)
}
