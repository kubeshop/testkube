package bus

import (
	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	nats "github.com/nats-io/nats.go"
)

func runServerOnPort(port int) *server.Server {
	opts := natsserver.DefaultTestOptions
	opts.JetStream = true
	opts.Port = port
	opts.Debug = true
	return natsserver.RunServer(&opts)
}

func RunServer() (*server.Server, string) {
	server := runServerOnPort(-1)
	return server, server.ClientURL()
}

func TestServerWithConnection() (*server.Server, *nats.Conn) {
	// given NATS server
	ns, natsUrl := RunServer()

	// and NATS connection
	nc, err := NewNATSConnection(natsUrl)
	if err != nil {
		panic(err)
	}

	return ns, nc
}
