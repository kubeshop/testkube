package adapter

import (
	"context"

	"github.com/kubeshop/testkube/pkg/logs/events"
)

// Adapter will listen to log chunks, and handle them based on log id (execution Id)
type Adapter interface {
	// Init will init for given id
	Init(ctx context.Context, id string) error
	// Notify will send data log events for particaular execution id
	Notify(ctx context.Context, id string, event events.Log) error
	// Stop will stop listening subscriber from sending data
	Stop(ctx context.Context, id string) error
	// Name subscriber name
	Name() string
}
