package event

import (
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

func TestEmbeddedServer_Start(t *testing.T) {

	ns, nc, err := ServerWithConnection(t.TempDir())
	assert.NoError(t, err)

	nc.Subscribe("events", func(msg *nats.Msg) {
		t.Logf("Received message: %s", string(msg.Data))
		ns.Shutdown()
	})

	nc.Publish("events", []byte("test"))

	t.Log("Waiting for shutdown")
	ns.WaitForShutdown()

}
