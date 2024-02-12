package state

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/event/bus"
)

func TestState(t *testing.T) {

	ns, nc := bus.TestServerWithConnection()
	defer ns.Shutdown()

	ctx := context.Background()

	js, err := jetstream.New(nc)
	assert.NoError(t, err)

	kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: "test-logsstae-bucket"})
	assert.NoError(t, err)

	s := NewState(kv)

	t.Run("get non existing state", func(t *testing.T) {
		state1, err := s.Get(ctx, "1")
		assert.NoError(t, err)
		assert.Equal(t, LogStateUnknown, state1)
	})

	t.Run("store state data and get it", func(t *testing.T) {
		err = s.Put(ctx, "1", LogStateFinished)
		assert.NoError(t, err)

		state1, err := s.Get(ctx, "1")
		assert.NoError(t, err)
		assert.Equal(t, LogStateFinished, state1)
	})

}
