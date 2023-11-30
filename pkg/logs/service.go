package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/consumer"
	"github.com/kubeshop/testkube/pkg/logs/events"
)

const (
	StreamPrefix = "log"

	StartSubject = "events.logs.start"
	StopSubject  = "events.logs.stop"

	StartQueue = "logsstart"
	StopQueue  = "logsstop"
)

func NewLogsService(nats *nats.EncodedConn, js jetstream.JetStream, httpAddress string) *LogsService {
	return &LogsService{
		nats:              nats,
		consumers:         []consumer.Consumer{},
		js:                js,
		log:               log.DefaultLogger.With("service", "logs-service"),
		Ready:             make(chan struct{}, 1),
		httpAddress:       httpAddress,
		consumerInstances: sync.Map{},
	}
}

type LogsService struct {
	log       *zap.SugaredLogger
	nats      *nats.EncodedConn
	js        jetstream.JetStream
	consumers []consumer.Consumer

	Ready chan struct{}

	// httpAddress is address for Kubernetes http health check handler
	httpAddress string

	// consumerInstances is internal executionID => consumer map which we need to clean
	// each pod can have different executionId set of consumers
	consumerInstances sync.Map
}

func (l *LogsService) AddConsumer(s consumer.Consumer) {
	l.consumers = append(l.consumers, s)
}

// RunHealthCheckHandler is a handler for health check events
// we need HTTP as GRPC probes starts from Kubernetes 1.25
func (l *LogsService) RunHealthCheckHandler(ctx context.Context) {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	l.log.Panic(http.ListenAndServe(l.httpAddress, nil))
}

func (l *LogsService) Run(ctx context.Context) (err error) {

	// LOGIC is like follows:

	// 1. Handle start stop events from nats
	//    assuming after start event something is pushing data to the stream
	//    it can be our handler or some other service

	l.log.Infow("starting logs service")

	// 2. For start event we must build stream for given execution id and start consuming it
	// this one will must a queue group each pod will get it's own
	l.nats.QueueSubscribe(StartSubject, StartQueue, func(event events.Trigger) {

		log := l.log.With("id", event.Id)

		s, err := l.CreateStream(ctx, event)
		if err != nil {
			l.log.Errorw("error creating stream", "error", err, "id", event.Id)
			return
		}

		log.Infow("stream created", "stream", s)

		streamName := StreamPrefix + event.Id

		// for each consumer create nats consumer and consume stream from it e.g. cloud s3 or others
		for i, consumer := range l.consumers {
			c, err := l.InitConsumer(ctx, consumer, streamName, event.Id, i)
			if err != nil {
				log.Errorw("error creating consumer", "consumer", consumer.Name(), "error", err)
				return
			}
			log.Infow("consumer created", "consumer", c.CachedInfo(), "stream", streamName)
			cons, err := c.Consume(l.HandleMessage(consumer, event))
			log.Infow("consumer started", "consumer", consumer.Name(), "id", event.Id, "stream", streamName)

			// store consumer instance so we can stop it later
			l.consumerInstances.Store(event.Id, cons)

			if err != nil {
				log.Errorw("error consuming", "error", err, "consumer", c.CachedInfo())
			}
		}
	})

	// listen on all pods as we don't control which one will have given consumer
	l.nats.Subscribe(StopSubject, func(event events.Trigger) {
		_, found := l.consumerInstances.LoadAndDelete(event.Id)
		if found {
			l.log.Infow("stopping consumer", "id", event.Id, "deleted", found)
			return
		}

		l.log.Debugw("consumer not found", "id", event.Id, "deleted", found)
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

func (l *LogsService) InitConsumer(ctx context.Context, consumer consumer.Consumer, streamName, id string, i int) (jetstream.Consumer, error) {
	name := fmt.Sprintf("lc%s%s%d", id, consumer.Name(), i)
	return l.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:    name,
		Durable: name,
		// FilterSubject: streamName,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
}

func (l *LogsService) CreateStream(ctx context.Context, event events.Trigger) (jetstream.Stream, error) {
	// create stream for incoming logs
	streamName := StreamPrefix + event.Id
	return l.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:    streamName,
		Storage: jetstream.FileStorage, // durable stream
	})
}

func (l *LogsService) HandleMessage(consumer consumer.Consumer, event events.Trigger) func(msg jetstream.Msg) {
	log := l.log.With("id", event.Id, "consumer", consumer.Name())

	return func(msg jetstream.Msg) {
		log.Infow("got message", "data", string(msg.Data()))

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
	}
}
