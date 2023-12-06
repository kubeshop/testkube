package logs

import (
	"context"
	"fmt"
	"testing"

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

	meta, err := client.Init(ctx)
	assert.NoError(t, err)
	assert.Equal(t, StreamPrefix+"111", meta.Name)

	err = client.PushBytes(ctx, []byte(`{"content":"hello 1"}`))
	assert.NoError(t, err)

	var startReceived, stopReceived bool

	_, err = nc.Subscribe(StartSubject, func(m *nats.Msg) {
		fmt.Printf("%s\n", m.Data)
		m.Respond([]byte("ok"))
		startReceived = true
	})
	assert.NoError(t, err)
	_, err = nc.Subscribe(StopSubject, func(m *nats.Msg) {
		fmt.Printf("%s\n", m.Data)
		m.Respond([]byte("ok"))
		stopReceived = true
	})

	assert.NoError(t, err)

	d, err := client.Start(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "ok", string(d.Message))

	d, err = client.Stop(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "ok", string(d.Message))

	assert.True(t, startReceived)
	assert.True(t, stopReceived)
}
