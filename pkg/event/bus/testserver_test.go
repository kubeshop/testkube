package bus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServerRestart(t *testing.T) {
	// given NATS server
	natsserver, natsUrl := RunServer()
	defer natsserver.Shutdown()

	// and NATS connection
	nc, err := NewNATSConnection(natsUrl)
	assert.NoError(t, err)

	sub, err := nc.SubscribeSync("aaa")
	assert.NoError(t, err)

	go func() {
		nc.Publish("aaa", []byte("hello"))
	}()

	msg, err := sub.NextMsg(100 * time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(msg.Data))
}
