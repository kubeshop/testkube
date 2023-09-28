package stream

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

const (
	StreamEndMessage = "<< Logs stream ended >>"

	streamName          = "logs"
	consumerName        = "logsconsumer"
	defaultTTL          = 2 * 24 * time.Hour
	allConsumerPrefix   = "lca"
	rangeConsumerPrefix = "lcr"
)

//go:generate mockgen -destination=./mock_stream.go -package=stream "github.com/kubeshop/testkube/pkg/executor/stream" LogsStream
type LogsStream interface {
	Init(ctx context.Context) error
	Publish(ctx context.Context, executionId string, line []byte) error
	End(ctx context.Context, executionId string) error
	GetRange(ctx context.Context, executionId string, from, count int) (chan []byte, error)
	Listen(ctx context.Context, executionId string) (chan []byte, error)
}

func NewJetstreamLogsStream(js jetstream.JetStream) JetstreamLogsStream {
	return JetstreamLogsStream{
		js: js,
		l:  log.DefaultLogger.With("service", "LogsStream"),
	}
}

type JetstreamLogsStream struct {
	js jetstream.JetStream
	l  *zap.SugaredLogger
}

func (c JetstreamLogsStream) Init(ctx context.Context) error {
	// Create a stream
	s, err := c.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{streamName + ".*"},
		MaxAge:   defaultTTL,
		Storage:  jetstream.FileStorage, // durable stream
	})

	if err == jetstream.ErrStreamNameAlreadyInUse {
		c.l.Debugw("stream already exists will use existing one", "stream", streamName)
	} else if err != nil {
		return err
	}

	info, _ := s.Info(ctx)
	c.l.Debugw("using stream", "stream", info)

	return nil
}

func (c JetstreamLogsStream) Publish(ctx context.Context, executionId string, line []byte) error {
	// INFO: first result var is publisher ACK can be used for only-once delivery (double ACK pattern)
	_, err := c.js.Publish(ctx, streamName+"."+executionId, line)
	return err
}

func (c JetstreamLogsStream) End(ctx context.Context, executionId string) error {
	_, err := c.js.Publish(ctx, streamName+"."+executionId, []byte(StreamEndMessage))
	return err
}

func (c JetstreamLogsStream) GetRange(ctx context.Context, executionId string, from, count int) (chan []byte, error) {
	ch := make(chan []byte)
	consumerName := rangeConsumerPrefix + executionId

	consumer, err := c.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: streamName + "." + executionId,
		DeliverPolicy: jetstream.DeliverByStartSequencePolicy,
		OptStartSeq:   uint64(from),
	})
	if err != nil {
		return ch, err
	}

	it, err := consumer.Messages()
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(ch)
		defer it.Stop()
		defer c.js.DeleteConsumer(ctx, streamName, consumerName)

		for i := 0; i < count; i++ {
			msg, err := it.Next()
			if err == jetstream.ErrMsgIteratorClosed {
				c.l.Debug("iterator closed")
				return
			} else if err != nil {
				return
			}
			msg.Ack()
			ch <- msg.Data()
		}
	}()
	return ch, nil
}

func (c JetstreamLogsStream) Listen(ctx context.Context, executionId string) (chan []byte, error) {
	ch := make(chan []byte)
	consumerName := allConsumerPrefix + executionId

	consumer, err := c.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: streamName + "." + executionId,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return ch, err
	}

	stopConsumer := make(chan struct{})

	cons, err := consumer.Consume(func(msg jetstream.Msg) {
		err := msg.Ack()
		if err != nil {
			c.l.Errorw("error acking message", "error", err, "msg", msg.Headers())
			return
		}
		d := msg.Data()

		if string(d) == StreamEndMessage {
			stopConsumer <- struct{}{}
			return
		}

		ch <- msg.Data()
	})

	go func() {
		<-stopConsumer
		cons.Stop()
		close(ch)
	}()

	if err != nil {
		return ch, err
	}

	return ch, nil
}
