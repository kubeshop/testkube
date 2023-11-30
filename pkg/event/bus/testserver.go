package bus

import (
	"os"

	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	nats "github.com/nats-io/nats.go"
)

func TestServerWithConnection() (*server.Server, *nats.Conn) {
	opts := &natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	opts.Debug = true

	dir, err := os.MkdirTemp("", "nats-*")
	if err != nil {
		panic(err)
	}
	opts.StoreDir = dir

	ns := natsserver.RunServer(opts)
	ns.EnableJetStream(&server.JetStreamConfig{
		StoreDir: dir,
	})

	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		panic(err)
	}

	return ns, nc
}
