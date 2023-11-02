package main

import (
	"context"

	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs"
	"github.com/kubeshop/testkube/pkg/logs/config"
	"github.com/kubeshop/testkube/pkg/logs/consumer"

	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	log := log.DefaultLogger.With("service", "logs")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Must(config.Get())

	// Event bus
	natsConn := Must(bus.NewNATSConnection(cfg.NatsURI))
	defer func() {
		log.Infof("closing nats connection")
		natsConn.Close()
	}()

	natsEncodedConn := Must(bus.NewNATSEncoddedConnection(cfg.NatsURI))
	defer func() {
		log.Infof("closing encoded nats connection")
		natsEncodedConn.Close()
	}()

	js := Must(jetstream.New(natsConn))

	svc := logs.NewLogsService(natsEncodedConn, js)
	svc.AddConsumer(consumer.NewDummyConsumer())
	svc.Run(ctx)
}

// Must helper function to panic on error
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
