package logs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/consumer"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

const (
	StreamName = "lg"
	StartTopic = "events.logs.start"
	StopTopic  = "events.logs.stop"
)

func NewLogsService(nats *nats.EncodedConn, js jetstream.JetStream) *LogsService {
	return &LogsService{
		nats:      nats,
		consumers: []consumer.Consumer{},
		js:        js,
		log:       log.DefaultLogger.With("service", "logs-service"),
		Ready:     make(chan struct{}, 1),
	}
}

type LogsService struct {
	log       *zap.SugaredLogger
	nats      *nats.EncodedConn
	js        jetstream.JetStream
	consumers []consumer.Consumer

	Ready chan struct{}
}

func (l *LogsService) AddConsumer(s consumer.Consumer) {
	l.consumers = append(l.consumers, s)
}

func (l *LogsService) Run(ctx context.Context) (err error) {

	// LOGIC is like follows:

	// 1. Handle start stop events from nats
	//    assuming after start event something is pushing data to the stream
	//    it can be our handler or some other service (like NAT beat)

	l.log.Infow("starting logs service")

	// TODO refactor abstract NATS logic from here?
	// TODO consider using durable topics for queue with Ack / Nack
	l.nats.QueueSubscribe("events.logs.stop", "startevents", func(event events.Trigger) {
		// TODO stop all consumers from consuming data for given execution id
	})

	// 2. For start event we must build stream for given execution id and start consuming it
	// this one will must a queue group each pod will get it's own
	l.nats.QueueSubscribe("events.logs.start", "startevents", func(event events.Trigger) {
		log := l.log.With("id", event.Id)

		// create stream for incoming logs
		streamName := StreamName + event.Id
		s, err := l.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
			Name:    streamName,
			Storage: jetstream.FileStorage, // durable stream
		})

		if err != nil {
			l.log.Errorw("error creating stream", "error", err, "id", event.Id, "stream", streamName)
			return
		}

		log.Infow("stream created", "stream", s)

		// for each consumer create nats consumer and consume stream from it e.g. cloud s3 or others
		for i, consumer := range l.consumers {
			name := fmt.Sprintf("lc_%s_%s_%d", event.Id, consumer.Name(), i)

			c, err := l.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
				Name:    name,
				Durable: name,
				// FilterSubject: streamName,
				DeliverPolicy: jetstream.DeliverAllPolicy,
			})

			log.Infow("consumer created", "consumer", c.CachedInfo(), "stream", streamName)

			if err != nil {
				log.Errorw("error creating consumer", "consumer", consumer.Name(), "error", err)
				return
			}

			cons, err := c.Consume(func(msg jetstream.Msg) {

				log.Infow("got message", "consumer", consumer.Name(), "id", event.Id, "data", string(msg.Data()))

				// deliver to subscriber
				logChunk := events.LogChunk{}
				json.Unmarshal(msg.Data(), &logChunk)
				err := consumer.Notify(event.Id, logChunk)

				if err != nil {
					if err := msg.Nak(); err != nil {
						log.Errorw("error nacking message", "error", err)
						return
					}
					return
				}

				if err := msg.Ack(); err != nil {
					log.Errorw("error acking message", "error", err)
				}
			})

			log.Infow("consumer started", "consumer", consumer.Name(), "id", event.Id, "stream", streamName)

			// TODO add `cons` and stop it on stop event
			var _ = cons

			if err != nil {
				log.Errorw("error consuming", "error", err, "consumer", c.CachedInfo())
			}
		}
	})

	l.Ready <- struct{}{}

	<-ctx.Done()

	// TODO
	// assuming this one will be scaled to multiple instances
	// how to handle pod issues here?
	// how to know that there is topic which is not handled by any subscriber?
	// we woudl need to check pending log topics and handle them

	// block

	return nil
}
