package bus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServerRestart(t *testing.T) {
	// given NATS server
	s, nc := TestServerWithConnection()
	defer s.Shutdown()

	// and NATS Subscription
	sub, err := nc.SubscribeSync("aaa")
	assert.NoError(t, err)

	// when data is published to NATS
	go func() {
		nc.Publish("aaa", []byte("hello"))
	}()

	// then we should be able to read it
	msg, err := sub.NextMsg(100 * time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(msg.Data))
}
