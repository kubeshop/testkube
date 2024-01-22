package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/kubeshop/testkube/pkg/logs/adapter"
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

	StopWaitTime = 60 * time.Second // when stop event is faster than first message arrived
)

type Consumer struct {
	// Context is a consumer context you can call Stop() method on it when no more messages are expected
	Context jetstream.ConsumeContext
	// Instance is a NATS consumer instance
	Instance jetstream.Consumer
}

func (ls *LogsService) initConsumer(ctx context.Context, a adapter.Adapter, streamName, id string, i int) (jetstream.Consumer, error) {
	name := fmt.Sprintf("lc%s%s%d", id, a.Name(), i)
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
func (ls *LogsService) handleMessage(a adapter.Adapter, event events.Trigger) func(msg jetstream.Msg) {
	log := ls.log.With("id", event.Id, "consumer", a.Name())

	return func(msg jetstream.Msg) {
		log.Infow("got message", "data", string(msg.Data()))

		// deliver to subscriber
		logChunk := events.Log{}
		err := json.Unmarshal(msg.Data(), &logChunk)
		if err != nil {
			if err := msg.Nak(); err != nil {
				log.Errorw("error nacking message", "error", err)
				return
			}
			return
		}

		err = a.Notify(event.Id, logChunk)
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
func (ls *LogsService) handleStart(ctx context.Context) func(msg *nats.Msg) {
	return func(msg *nats.Msg) {
		event := events.Trigger{}
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			ls.log.Errorw("can't handle start event", "error", err)
			return
		}
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

		// reply to start event that everything was initialized correctly
		err = msg.Respond([]byte("ok"))
		if err != nil {
			log.Errorw("error responding to start event", "error", err)
			return
		}
	}

}

// handleStop will handle stop event and stop logs consumers, also clean consumers state
func (ls *LogsService) handleStop(ctx context.Context) func(msg *nats.Msg) {
	return func(msg *nats.Msg) {
		ls.log.Debugw("got stop event")
		time.Sleep(StopWaitTime)

		event := events.Trigger{}
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			ls.log.Errorw("can't handle stop event", "error", err)
			return
		}

		l := ls.log.With("id", event.Id, "event", "stop")

		maxTries := 10
		repeated := 0

		toDelete := []string{}
		deleted := false
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
					l.Debugw("consumer not found on this pod", "found", found, "name", name)
					toDelete = append(toDelete[:i], toDelete[i+1:]...)
					goto loop // rewrite toDelete and start again
				}

				consumer := c.(Consumer)

				info, err := consumer.Instance.Info(ctx)
				if err != nil {
					l.Errorw("error getting consumer info", "error", err, "id", event.Id)
					continue
				}

				// finally delete consumer
				if info.NumPending == 0 {
					if !deleted {
						deleted = true
					}
					consumer.Context.Stop()
					ls.consumerInstances.Delete(name)
					toDelete = append(toDelete[:i], toDelete[i+1:]...)
					l.Infow("stopping consumer", "id", name)
					goto loop // rewrite toDelete and start again
				}
			}

			if len(toDelete) == 0 && !deleted {
				l.Debugw("no consumers on this pod registered for id", "id", event.Id)
				return
			} else if len(toDelete) == 0 {
				ls.state.Put(ctx, event.Id, state.LogStateFinished)
				l.Infow("execution logs consumers stopped", "id", event.Id)
				err = msg.Respond([]byte("stopped"))
				if err != nil {
					l.Errorw("error responding to stop event", "error", err)
					return
				}
				return
			}

			// handle max tries of cleaning executors
			repeated++
			if repeated >= maxTries {
				l.Errorw("error cleaning consumeres after max tries", "toDeleteLeft", toDelete, "tries", repeated)
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
