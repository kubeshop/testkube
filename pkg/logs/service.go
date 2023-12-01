package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/consumer"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/state"
)

const (
	StatusPending  byte = 1
	StatusFinished byte = 2

	StreamPrefix = "log"

	StartSubject = "events.logs.start"
	StopSubject  = "events.logs.stop"

	StartQueue = "logsstart"
	StopQueue  = "logsstop"

	BuncketName = "logsState"
)

func NewLogsService(nats *nats.EncodedConn, js jetstream.JetStream, state state.Interface, httpAddress string) *LogsService {
	return &LogsService{
		nats:              nats,
		adapters:          []consumer.Adapter{},
		js:                js,
		log:               log.DefaultLogger.With("service", "logs-service"),
		Ready:             make(chan struct{}, 1),
		httpAddress:       httpAddress,
		consumerInstances: sync.Map{},
		state:             state,
	}
}

type LogsService struct {
	log      *zap.SugaredLogger
	nats     *nats.EncodedConn
	js       jetstream.JetStream
	adapters []consumer.Adapter

	Ready chan struct{}

	// httpAddress is address for Kubernetes http health check handler
	httpAddress string
	httpServer  *http.Server

	// consumerInstances is internal executionID => consumer map which we need to clean
	// each pod can have different executionId set of consumers
	consumerInstances sync.Map

	// state manager for keeping logs state (pending, finished)
	// will allow to distiguish from where load data from in OSS
	// cloud will be loading always them locally
	state state.Interface
}

// AddAdapter adds new adapter to logs service adapters will be configred based on given mode
// e.g. cloud mode will get cloud adapter to store logs directly on the cloud
func (l *LogsService) AddAdapter(a consumer.Adapter) {
	l.adapters = append(l.adapters, a)
}

// RunHealthCheckHandler is a handler for health check events
// we need HTTP as GRPC probes starts from Kubernetes 1.25
func (l *LogsService) RunHealthCheckHandler(ctx context.Context) error {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	l.httpServer = &http.Server{
		Addr: ":8080",
	}

	l.log.Infow("starting health check handler", "address", l.httpAddress)
	return l.httpServer.ListenAndServe()
}

func (l *LogsService) Shutdown(ctx context.Context) (err error) {
	err = l.httpServer.Shutdown(ctx)
	if err != nil {
		return err
	}

	// TODO decide how to handle graceful shutdown of consumers

	return nil
}

type Consumer struct {
	// Context is a consumer context you can call Stop() method on it when no more messages are expected
	Context jetstream.ConsumeContext
	// Instance is a NATS consumer instance
	Instance jetstream.Consumer
}

func (l *LogsService) Run(ctx context.Context) (err error) {
	// 1. Handle start and stop events from nats
	//    assuming after start event something is pushing data to the stream
	//    it can be our handler or some other service

	l.log.Infow("starting logs service")

	// 2. For start event we must build stream for given execution id and start consuming it
	// this one will must follow a queue group each pod will get it's own bunch of executions to handle
	l.nats.QueueSubscribe(StartSubject, StartQueue, func(event events.Trigger) {
		l.state.Put(ctx, event.Id, state.LogStatePending)
		log := l.log.With("id", event.Id)
		s, err := l.CreateStream(ctx, event)
		if err != nil {
			l.log.Errorw("error creating stream", "error", err, "id", event.Id)
			return
		}

		log.Infow("stream created", "stream", s)

		streamName := StreamPrefix + event.Id

		// for each adapter create NATS consumer and consume stream from it e.g. cloud s3 or others
		for i, adapter := range l.adapters {
			c, err := l.InitConsumer(ctx, adapter, streamName, event.Id, i)
			if err != nil {
				log.Errorw("error creating consumer", "consumer", adapter.Name(), "error", err)
				return
			}
			log.Infow("consumer created", "consumer", c.CachedInfo(), "stream", streamName)
			cons, err := c.Consume(l.HandleMessage(adapter, event))
			log.Infow("consumer started", "consumer", adapter.Name(), "id", event.Id, "stream", streamName)

			// store consumer instance so we can stop it later in StopSubject handler
			l.consumerInstances.Store(event.Id+"_"+adapter.Name(), Consumer{
				Context:  cons,
				Instance: c,
			})

			if err != nil {
				log.Errorw("error consuming", "error", err, "consumer", c.CachedInfo())
			}
		}
	})

	// listen on all pods as we don't control which one will have given consumer
	// Stop event will be triggered by logs process controller (scheduler)
	l.nats.Subscribe(StopSubject, func(event events.Trigger) {
		toDelete := []string{}
		for _, consumer := range l.adapters {
			toDelete = append(toDelete, event.Id+"_"+consumer.Name())
		}

		consumerDeleteWaitInterval := 5 * time.Second
		for {
			// Delete each consumer for given execution id
			for _, name := range toDelete {
				// load consumer and check if has pending messages
				c, found := l.consumerInstances.Load(name)
				if !found {
					l.log.Errorw("error getting consumer", "found", found, "id", event.Id)
					continue
				}

				consumer := c.(Consumer)

				info, err := consumer.Instance.Info(ctx)
				if err != nil {
					l.log.Errorw("error getting consumer info", "error", err, "id", event.Id)
					continue
				}

				// finally delete consumer
				if info.NumPending == 0 {
					consumer.Context.Stop()
					l.consumerInstances.Delete(name)
					l.log.Infow("stopping consumer", "id", name)
					continue
				}
			}

			if len(toDelete) == 0 {
				l.state.Put(ctx, event.Id, state.LogStateFinished)
				l.log.Infow("all logs consumers stopped", "id", event.Id)
				return
			}

			time.Sleep(consumerDeleteWaitInterval)
		}
	})

	l.Ready <- struct{}{}

	// block
	<-ctx.Done()

	// TODO how to handle pod issues here?
	// TODO how to know that there is topic which is not handled by any subscriber?
	// TODO we woudl need to check pending log topics and handle them after restart in case of log pod disaster

	return nil
}

func (l *LogsService) InitConsumer(ctx context.Context, consumer consumer.Adapter, streamName, id string, i int) (jetstream.Consumer, error) {
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

func (l *LogsService) HandleMessage(consumer consumer.Adapter, event events.Trigger) func(msg jetstream.Msg) {
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
