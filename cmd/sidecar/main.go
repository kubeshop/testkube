package main

import (
	"context"

	"github.com/kubeshop/testkube/pkg/event/bus"
	"github.com/kubeshop/testkube/pkg/k8sclient"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/client"
	"github.com/kubeshop/testkube/pkg/logs/config"
	"github.com/kubeshop/testkube/pkg/logs/sidecar"
	"github.com/kubeshop/testkube/pkg/ui"

	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	log := log.DefaultLogger.With("service", "logs-sidecar")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Must(config.Get())

	// Event bus
	nc := Must(bus.NewNATSConnection(bus.ConnectionConfig{
		NatsURI:            cfg.NatsURI,
		NatsSecure:         cfg.NatsSecure,
		NatsSkipVerify:     cfg.NatsSkipVerify,
		NatsCertFile:       cfg.NatsCertFile,
		NatsKeyFile:        cfg.NatsKeyFile,
		NatsCAFile:         cfg.NatsCAFile,
		NatsConnectTimeout: cfg.NatsConnectTimeout,
	}))
	defer func() {
		log.Infof("closing nats connection")
		nc.Close()
	}()

	js := Must(jetstream.New(nc))

	clientset, err := k8sclient.ConnectToK8s()
	if err != nil {
		ui.ExitOnError("Creating k8s clientset", err)
		return
	}

	podsClient := clientset.CoreV1().Pods(cfg.Namespace)

	logsStream, err := client.NewNatsLogStream(nc)
	if err != nil {
		ui.ExitOnError("error creating logs stream", err)
		return
	}

	// run Sidecar Logs Proxy - it will proxy logs from pod to nats
	proxy := sidecar.NewProxy(clientset, podsClient, logsStream, js, log, cfg.Namespace, cfg.ExecutionId, cfg.Source)
	if err := proxy.Run(ctx); err != nil {
		log.Errorw("error proxying logs", "error", err)
	}
}

// Must helper function to panic on error
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
