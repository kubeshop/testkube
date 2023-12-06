package state

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"
)

// KV is type implicitly interfaced from NATS
type NatsKV interface {
	Get(ctx context.Context, key string) (jetstream.KeyValueEntry, error)
	Put(ctx context.Context, key string, value []byte) (uint64, error)
}
