package logs

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

const (
	StreamName          = "LOGS"
	ConsumerName        = "LOGSCONSUMER"
	defaultTTL          = 2 * 24 * time.Hour
	allConsumerPrefix   = "lca"
	rangeConsumerPrefix = "lcr"
)

type LogsCache interface {
	Init(ctx context.Context) error
	Publish(ctx context.Context, executionId string, line []byte) error
	GetRange(ctx context.Context, executionId string, from, count int) (chan []byte, error)
	Listen(ctx context.Context, executionId string) (chan []byte, error)
}

func NewJetstreamLogsCache(js jetstream.JetStream) JetstreamLogsCache {
	return JetstreamLogsCache{
		js: js,
		l:  log.DefaultLogger.With("service", "LogsCache"),
	}
}

type JetstreamLogsCache struct {
	js jetstream.JetStream
	l  *zap.SugaredLogger
}

func (c JetstreamLogsCache) Init(ctx context.Context) error {
	// Create a stream
	s, err := c.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     StreamName,
		Subjects: []string{StreamName + ".*"},
		MaxAge:   defaultTTL,
		Storage:  jetstream.FileStorage, // durable stream
	})

	if err == jetstream.ErrStreamNameAlreadyInUse {
		c.l.Warnw("stream already exists", "stream", StreamName)
	} else if err != nil {
		return err
	} else {
		info, _ := s.Info(ctx)
		c.l.Debugw("created stream", "stream", info)
	}

	return nil
}

func (c JetstreamLogsCache) Publish(ctx context.Context, executionId string, line []byte) error {
	// INFO: first result var is publisher ACK can be used for only-once delivery (double ACK pattern)
	_, err := c.js.Publish(ctx, StreamName+"."+executionId, line)
	return err
}

func (c JetstreamLogsCache) GetRange(ctx context.Context, executionId string, from, count int) (chan []byte, error) {
	ch := make(chan []byte)
	consumerName := rangeConsumerPrefix + executionId

	consumer, err := c.js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: StreamName + "." + executionId,
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
		defer c.js.DeleteConsumer(ctx, StreamName, consumerName)

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

func (c JetstreamLogsCache) Listen(ctx context.Context, executionId string) (chan []byte, error) {
	ch := make(chan []byte)
	consumerName := allConsumerPrefix + executionId

	consumer, err := c.js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: StreamName + "." + executionId,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return ch, err
	}

	cons, err := consumer.Consume(func(msg jetstream.Msg) {
		err := msg.Ack()
		if err != nil {
			return
		}
		ch <- msg.Data()
	})

	defer cons.Stop()

	if err != nil {
		return ch, err
	}

	return ch, nil
}
