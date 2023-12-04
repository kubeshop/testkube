package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/pkg/logs/consumer"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/state"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	StatusPending  byte = 1
	StatusFinished byte = 2

	StreamPrefix = "log"

	StartSubject = "events.logs.start"
	StopSubject  = "events.logs.stop"

	StartQueue = "logsstart"
	StopQueue  = "logsstop"
)

type Consumer struct {
	// Context is a consumer context you can call Stop() method on it when no more messages are expected
	Context jetstream.ConsumeContext
	// Instance is a NATS consumer instance
	Instance jetstream.Consumer
}

func (ls *LogsService) initConsumer(ctx context.Context, consumer consumer.Adapter, streamName, id string, i int) (jetstream.Consumer, error) {
	name := fmt.Sprintf("lc%s%s%d", id, consumer.Name(), i)
	return ls.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:    name,
		Durable: name,
		// FilterSubject: streamName,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
}

func (ls *LogsService) createStream(ctx context.Context, event events.Trigger) (jetstream.Stream, error) {
	// create stream for incoming logs
	streamName := StreamPrefix + event.Id
	return ls.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:    streamName,
		Storage: jetstream.FileStorage, // durable stream as we can hit mem limit
	})
}

// handleMessage will handle incoming message from logs stream and proxy it to given adapter
func (ls *LogsService) handleMessage(adapter consumer.Adapter, event events.Trigger) func(msg jetstream.Msg) {
	log := ls.log.With("id", event.Id, "consumer", adapter.Name())

	return func(msg jetstream.Msg) {
		log.Infow("got message", "data", string(msg.Data()))

		// deliver to subscriber
		logChunk := events.LogChunk{}
		json.Unmarshal(msg.Data(), &logChunk)
		err := adapter.Notify(event.Id, logChunk)

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

// handleStart will handle start event and create logs consumers, also manage state of given (execution) id
func (ls *LogsService) handleStart(ctx context.Context) func(event events.Trigger) {
	return func(event events.Trigger) {
		log := ls.log.With("id", event.Id, "event", "start")

		ls.state.Put(ctx, event.Id, state.LogStatePending)
		s, err := ls.createStream(ctx, event)
		if err != nil {
			ls.log.Errorw("error creating stream", "error", err, "id", event.Id)
			return
		}

		log.Infow("stream created", "stream", s)

		streamName := StreamPrefix + event.Id

		// for each adapter create NATS consumer and consume stream from it e.g. cloud s3 or others
		for i, adapter := range ls.adapters {
			l := log.With("adapter", adapter.Name())
			c, err := ls.initConsumer(ctx, adapter, streamName, event.Id, i)
			if err != nil {
				log.Errorw("error creating consumer", "error", err)
				return
			}

			// handle message per each adapter
			l.Infow("consumer created", "consumer", c.CachedInfo(), "stream", streamName)
			cons, err := c.Consume(ls.handleMessage(adapter, event))
			if err != nil {
				log.Errorw("error creating consumer", "error", err, "consumer", c.CachedInfo())
				continue
			}

			// store consumer instance so we can stop it later in StopSubject handler
			ls.consumerInstances.Store(event.Id+"_"+adapter.Name(), Consumer{
				Context:  cons,
				Instance: c,
			})

			l.Infow("consumer started", "consumer", adapter.Name(), "id", event.Id, "stream", streamName)
		}
	}
}

// handleStop will handle stop event and stop logs consumers, also clean consumers state
func (ls *LogsService) handleStop(ctx context.Context) func(event events.Trigger) {
	return func(event events.Trigger) {

		l := ls.log.With("id", event.Id, "event", "stop")

		maxRepeat := 10
		repeated := 0

		toDelete := []string{}
		for _, adapter := range ls.adapters {
			toDelete = append(toDelete, event.Id+"_"+adapter.Name())
		}

		consumerDeleteWaitInterval := 5 * time.Second

		for {
		loop:
			// Delete each consumer for given execution id
			for i, name := range toDelete {
				// load consumer and check if has pending messages
				c, found := ls.consumerInstances.Load(name)
				if !found {
					l.Warnw("consumer not found", "found", found, "name", name)
					continue
				}

				consumer := c.(Consumer)

				info, err := consumer.Instance.Info(ctx)
				if err != nil {
					l.Errorw("error getting consumer info", "error", err, "id", event.Id)
					continue
				}

				// finally delete consumer
				if info.NumPending == 0 {
					consumer.Context.Stop()
					ls.consumerInstances.Delete(name)
					toDelete = append(toDelete[:i], toDelete[i+1:]...)
					l.Infow("stopping consumer", "id", name)
					goto loop // rewrite toDelete and start again
				}
			}

			if len(toDelete) == 0 {
				ls.state.Put(ctx, event.Id, state.LogStateFinished)
				l.Infow("all logs consumers stopped", "id", event.Id)
				return
			}

			// handle max tries of cleaning executors
			repeated++
			if repeated >= maxRepeat {
				l.Errorw("error cleaning consumeres", "toDeleteLeft", toDelete)
				return
			}

			time.Sleep(consumerDeleteWaitInterval)
		}
	}
}

type ConsumerStats struct {
	Count int
	Names []string
}

func (ls *LogsService) GetConsumersStats(ctx context.Context) (stats ConsumerStats) {

	ls.consumerInstances.Range(func(key, value interface{}) bool {
		stats.Count++
		stats.Names = append(stats.Names, key.(string))
		return true
	})

	return
}
