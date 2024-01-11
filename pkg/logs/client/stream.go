package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/utils"
)

func NewNatsLogStream(nc *nats.Conn, id string) (Stream, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return &NatsLogStream{}, err
	}

	return &NatsLogStream{
		nc:         nc,
		js:         js,
		log:        log.DefaultLogger,
		id:         id,
		streamName: StreamPrefix + id,
	}, nil
}

type NatsLogStream struct {
	nc         *nats.Conn
	js         jetstream.JetStream
	log        *zap.SugaredLogger
	streamName string
	id         string
}

func (c NatsLogStream) Init(ctx context.Context) (StreamMetadata, error) {
	s, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:    c.streamName,
		Storage: jetstream.FileStorage, // durable stream
	})

	if err == nil {
		c.log.Debugw("stream upserted", "info", s.CachedInfo())
	}

	return StreamMetadata{Name: c.streamName}, err

}

// Push log chunk to NATS stream
func (c NatsLogStream) Push(ctx context.Context, chunk events.Log) error {
	b, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	return c.PushBytes(ctx, b)
}

// Push log chunk to NATS stream
// TODO handle message repeat with backoff strategy on error
func (c NatsLogStream) PushBytes(ctx context.Context, chunk []byte) error {
	_, err := c.js.Publish(ctx, c.streamName, chunk)
	return err
}

// Start emits start event to the stream - logs service will handle start and create new stream
func (c NatsLogStream) Start(ctx context.Context) (resp StreamResponse, err error) {
	return c.syncCall(ctx, StartSubject)
}

// Stop emits stop event to the stream and waits for given stream to be stopped fully - logs service will handle stop and close stream and all subscribers
func (c NatsLogStream) Stop(ctx context.Context) (resp StreamResponse, err error) {
	return c.syncCall(ctx, StopSubject)
}

// Get returns channel with log stream chunks for given execution id connects through GRPC to log service
func (c NatsLogStream) Get(ctx context.Context) (chan events.LogResponse, error) {
	ch := make(chan events.LogResponse)

	name := fmt.Sprintf("lc%s%s", c.id, utils.RandAlphanum(6))
	cons, err := c.js.CreateOrUpdateConsumer(ctx, c.streamName, jetstream.ConsumerConfig{
		Name:          name,
		Durable:       name,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})

	if err != nil {
		return ch, err
	}

	log := c.log.With("id", c.id)
	cons.Consume(func(msg jetstream.Msg) {
		log.Debugw("got message", "data", string(msg.Data()))

		// deliver to subscriber
		logChunk := events.Log{}
		err := json.Unmarshal(msg.Data(), &logChunk)
		if err != nil {
			if err := msg.Nak(); err != nil {
				log.Errorw("error nacking message", "error", err)
				ch <- events.LogResponse{Error: err}
				return
			}
			return
		}

		if err := msg.Ack(); err != nil {
			ch <- events.LogResponse{Error: err}
			log.Errorw("error acking message", "error", err)
			return
		}

		ch <- events.LogResponse{Log: logChunk}
	})

	return ch, nil

}

// syncCall sends request to given subject and waits for response
func (c NatsLogStream) syncCall(ctx context.Context, subject string) (resp StreamResponse, err error) {
	b, _ := json.Marshal(events.Trigger{Id: c.id})
	m, err := c.nc.Request(subject, b, time.Minute)
	if err != nil {
		return resp, err
	}

	return StreamResponse{Message: m.Data}, nil
}