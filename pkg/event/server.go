package event

import (
	"fmt"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

const natsConnectionTimeout = 10 * time.Second

var ErrNatsEmbeddedServerTimeout = fmt.Errorf("server not ready for connections in %s", natsConnectionTimeout)

// ServerWithConnection starts NATS server with embedded JetStream, wait for readines and returns connection to it
func ServerWithConnection(dir string) (*server.Server, *nats.Conn, error) {
	opts := &server.Options{
		JetStream:             true,
		Port:                  4222,
		Host:                  "localhost",
		StoreDir:              dir,
		NoLog:                 false,
		NoSigs:                true,
		MaxControlLine:        4096,
		DisableShortFirstPing: true,
	}

	// Initialize new server with options
	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, nil, err
	}

	// Start the server via goroutine
	ns.Start()
	ns.EnableJetStream(&server.JetStreamConfig{
		StoreDir: dir,
	})

	// Wait for server to be ready for connections - this one will block
	if !ns.ReadyForConnections(natsConnectionTimeout) {
		return nil, nil, ErrNatsEmbeddedServerTimeout
	}

	// Connect to server
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		return nil, nil, err
	}

	return ns, nc, nil
}
