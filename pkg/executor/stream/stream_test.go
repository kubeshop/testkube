//go:build e2e

package stream

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
)

func TestConsume_ConsumesUntilEndOfStream(t *testing.T) {

	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nc, _ := nats.Connect(nats.DefaultURL)

	// Create a JetStream management interface
	js, _ := jetstream.New(nc)

	s := NewJetstreamLogsStream(js)
	s.Init(ctx)

	err = s.Publish(ctx, "test1", []byte("msg1"))
	assert.NoError(t, err)
	err = s.Publish(ctx, "test1", []byte("msg2"))
	assert.NoError(t, err)
	err = s.Publish(ctx, "test1", []byte("msg3"))
	assert.NoError(t, err)
	err = s.Publish(ctx, "test1", []byte("msg4"))
	assert.NoError(t, err)
	err = s.End(ctx, "test1")
	assert.NoError(t, err)

	ch, err := s.Listen(ctx, "test1")
	assert.NoError(t, err)
	var count int
	for range ch {
		count++
	}

	assert.Equal(t, 4, count)
}
