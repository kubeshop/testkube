package logs

import (
	"context"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type Stream interface {
	Init(ctx context.Context) error
	Push(ctx context.Context, chunk []byte) error
}

const proxyStreamPrefix = "lg"

func NewNATSStream(js jetstream.JetStream, id string) Stream {
	return NATSStream{
		js:         js,
		log:        log.DefaultLogger,
		streamName: proxyStreamPrefix + id,
	}
}

type NATSStream struct {
	js         jetstream.JetStream
	log        *zap.SugaredLogger
	streamName string
}

func (c NATSStream) Init(ctx context.Context) error {
	s, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:    c.streamName,
		Storage: jetstream.FileStorage, // durable stream
	})

	c.log.Debugw("logs proxy stream upserted", "info", s.CachedInfo())

	return err

}

// Push log chunk to NATS stream
// TODO handle message repeat with backoff strategy on error
func (c NATSStream) Push(ctx context.Context, chunk []byte) error {
	_, err := c.js.Publish(ctx, c.streamName, chunk)
	return err
}
