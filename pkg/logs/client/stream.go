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

const (
	ConsumerPrefix          = "lc"
	defaultLogMessageMaxAge = time.Hour * 24 * 2
)

func NewNatsLogStream(nc *nats.Conn) (s Stream, err error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return s, err
	}

	return &NatsLogStream{
		nc:     nc,
		js:     js,
		log:    log.DefaultLogger,
		maxAge: defaultLogMessageMaxAge,
	}, nil
}

type NatsLogStream struct {
	nc  *nats.Conn
	js  jetstream.JetStream
	log *zap.SugaredLogger
	// log message maxAge time
	maxAge time.Duration
}

func (c NatsLogStream) Init(ctx context.Context, id string) (StreamMetadata, error) {
	s, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:    c.Name(id),
		Storage: jetstream.FileStorage, // durable stream
		MaxAge:  c.maxAge,
	})

	if err == nil {
		c.log.Debugw("stream upserted", "info", s.CachedInfo())
	}

	return StreamMetadata{Name: c.Name(id)}, err

}

func (c NatsLogStream) Finish(ctx context.Context, id string) error {
	return c.Push(ctx, id, events.NewFinishLog())
}

// Push log chunk to NATS stream
func (c NatsLogStream) Push(ctx context.Context, id string, log *events.Log) error {
	b, err := json.Marshal(log)
	if err != nil {
		return err
	}
	return c.PushBytes(ctx, id, b)
}

// Push log chunk to NATS stream
// TODO handle message repeat with backoff strategy on error
func (c NatsLogStream) PushBytes(ctx context.Context, id string, bytes []byte) error {
	_, err := c.js.Publish(ctx, c.Name(id), bytes)
	return err
}

// Start emits start event to the stream - logs service will handle start and create new stream
func (c NatsLogStream) Start(ctx context.Context, id string) (resp StreamResponse, err error) {
	return c.syncCall(ctx, StartSubject, id)
}

// Stop emits stop event to the stream and waits for given stream to be stopped fully - logs service will handle stop and close stream and all subscribers
func (c NatsLogStream) Stop(ctx context.Context, id string) (resp StreamResponse, err error) {
	return c.syncCall(ctx, StopSubject, id)
}

// Get returns channel with log stream chunks for given execution id connects through GRPC to log service
func (c NatsLogStream) Get(ctx context.Context, id string) (chan events.LogResponse, error) {
	ch := make(chan events.LogResponse)

	name := fmt.Sprintf("%s%s%s", ConsumerPrefix, id, utils.RandAlphanum(6))
	cons, err := c.js.CreateOrUpdateConsumer(
		ctx,
		c.Name(id),
		jetstream.ConsumerConfig{
			Name:          name,
			Durable:       name,
			DeliverPolicy: jetstream.DeliverAllPolicy,
		},
	)

	if err != nil {
		return ch, err
	}

	log := c.log.With("id", id)

	go func() {
		defer close(ch)
		for {
			msg, err := cons.Next()
			if err != nil {
				ch <- events.LogResponse{Error: err}
				return
			}

			if finished := c.handleJetstreamMessage(log, ch, msg); finished {
				return
			}
		}
	}()

	return ch, nil
}

func (c NatsLogStream) handleJetstreamMessage(log *zap.SugaredLogger, ch chan events.LogResponse, msg jetstream.Msg) (finish bool) {
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

	if events.IsFinished(&logChunk) {
		return true
	}

	ch <- events.LogResponse{Log: logChunk}

	return
}

// syncCall sends request to given subject and waits for response
func (c NatsLogStream) syncCall(ctx context.Context, subject, id string) (resp StreamResponse, err error) {
	b, err := json.Marshal(events.NewTrigger(id))
	if err != nil {
		return resp, err
	}
	m, err := c.nc.Request(subject, b, time.Minute)
	if err != nil {
		return resp, err
	}

	return StreamResponse{Message: m.Data}, nil
}

func (c NatsLogStream) Name(id ...string) string {
	if len(id) > 0 {
		return StreamPrefix + id[0]
	}

	return StreamPrefix + utils.RandAlphanum(10)
}
