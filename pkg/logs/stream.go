package logs

import (
	"context"
	"encoding/json"
	"time"

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

type StreamMetadata struct {
	Name string
}

type StreamInitializer interface {
	// Init creates or updates stream on demand
	Init(ctx context.Context) (meta StreamMetadata, err error)
}

type StreamPusher interface {
	// Push sends logs to log stream
	Push(ctx context.Context, chunk events.LogChunk) error
	// PushBytes sends RAW bytes to log stream, developer is responsible for marshaling valid data
	PushBytes(ctx context.Context, chunk []byte) error
}

type StreamResponse struct {
	Message []byte
	Error   bool
}

type StreamTrigger interface {
	// Trigger start event
	Start(ctx context.Context) (StreamResponse, error)
	// Trigger stop event
	Stop(ctx context.Context) (StreamResponse, error)
}

func NewLogsStream(nc *nats.Conn, id string) (Stream, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return LogsStream{}, err
	}

	return LogsStream{
		nc:         nc,
		js:         js,
		log:        log.DefaultLogger,
		id:         id,
		streamName: StreamPrefix + id,
	}, nil
}

type LogsStream struct {
	nc         *nats.Conn
	js         jetstream.JetStream
	log        *zap.SugaredLogger
	streamName string
	id         string
}

func (c LogsStream) Init(ctx context.Context) (StreamMetadata, error) {
	s, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:    c.streamName,
		Storage: jetstream.FileStorage, // durable stream
	})

	c.log.Debugw("stream upserted", "info", s.CachedInfo())

	return StreamMetadata{Name: c.streamName}, err

}

// Push log chunk to NATS stream
func (c LogsStream) Push(ctx context.Context, chunk events.LogChunk) error {
	b, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	return c.PushBytes(ctx, b)
}

// Push log chunk to NATS stream
// TODO handle message repeat with backoff strategy on error
func (c LogsStream) PushBytes(ctx context.Context, chunk []byte) error {
	_, err := c.js.Publish(ctx, c.streamName, chunk)
	return err
}

// Start emits start event to the stream - logs service will handle start and create new stream
func (c LogsStream) Start(ctx context.Context) (resp StreamResponse, err error) {
	return c.syncCall(ctx, StartSubject)
}

// Stop emits stop event to the stream and waits for given stream to be stopped fully - logs service will handle stop and close stream and all subscribers
func (c LogsStream) Stop(ctx context.Context) (resp StreamResponse, err error) {
	return c.syncCall(ctx, StopSubject)
}

// syncCall sends request to given subject and waits for response
func (c LogsStream) syncCall(ctx context.Context, subject string) (resp StreamResponse, err error) {
	b, _ := json.Marshal(events.Trigger{Id: c.id})
	m, err := c.nc.Request(subject, b, time.Minute)
	if err != nil {
		return resp, err
	}

	return StreamResponse{Message: m.Data}, nil
}