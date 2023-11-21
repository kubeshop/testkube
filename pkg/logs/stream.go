package logs

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
)

type Stream interface {
	StreamInitializer
	StreamPusher
	StreamTrigger
}

type StreamInitializer interface {
	// Init creates or updates stream on demand
	Init(ctx context.Context) error
}

type StreamPusher interface {
	// Push sends logs to log stream
	Push(ctx context.Context, chunk events.LogChunk) error
	// PushBytes sends RAW bytes to log stream, developer is responsible for marshaling valid data
	PushBytes(ctx context.Context, chunk []byte) error
}

type StreamTrigger interface {
	// Trigger start / stop events
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

func NewNATSStream(nc *nats.Conn, id string) (Stream, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return NATSStream{}, err
	}

	return NATSStream{
		nc:         nc,
		js:         js,
		log:        log.DefaultLogger,
		id:         id,
		streamName: StreamPrefix + id,
	}, nil
}

type NATSStream struct {
	nc         *nats.Conn
	js         jetstream.JetStream
	log        *zap.SugaredLogger
	streamName string
	id         string
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
func (c NATSStream) Push(ctx context.Context, chunk events.LogChunk) error {
	b, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	_, err = c.js.Publish(ctx, c.streamName, b)
	return err
}

// Push log chunk to NATS stream
// TODO handle message repeat with backoff strategy on error
func (c NATSStream) PushBytes(ctx context.Context, chunk []byte) error {
	_, err := c.js.Publish(ctx, c.streamName, chunk)
	return err
}

func (c NATSStream) Start(ctx context.Context) error {
	event := events.Trigger{Id: c.id}
	b, _ := json.Marshal(event)
	return c.nc.Publish(StartSubject, b)
}

func (c NATSStream) Stop(ctx context.Context) error {
	event := events.Trigger{Id: c.id}
	b, _ := json.Marshal(event)
	return c.nc.Publish(StopSubject, b)
}
