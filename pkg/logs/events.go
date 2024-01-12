package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
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
)

type Consumer struct {
	// Name of the consumer
	Name string
	// Context is a consumer context you can call Stop() method on it when no more messages are expected
	Context jetstream.ConsumeContext
	// Instance is a NATS consumer instance
	Instance jetstream.Consumer
}

func (ls *LogsService) initConsumer(ctx context.Context, a adapter.Adapter, streamName, id string, i int) (jetstream.Consumer, error) {
	name := fmt.Sprintf("lc%s%s%d", id, a.Name(), i)
	return ls.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name: name,
		// Durable: name,
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
		log.Debugw("got message", "data", string(msg.Data()))

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
				Name:     event.Id + "_" + adapter.Name(),
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
		var (
			wg      sync.WaitGroup
			stopped = 0
			event   = events.Trigger{}
		)

		ls.log.Debugw("got stop event")

		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			ls.log.Errorw("can't handle stop event", "error", err)
			return
		}

		l := ls.log.With("id", event.Id, "event", "stop")

		err = msg.Respond([]byte("stop-queued"))
		if err != nil {
			l.Errorw("error responding to stop event", "error", err)
		}

		for _, adapter := range ls.adapters {
			consumerName := event.Id + "_" + adapter.Name()

			// locate consumer on this pod
			c, found := ls.consumerInstances.Load(consumerName)
			l.Debugw("consumer instance", "c", c, "found", found, "name", consumerName)
			if !found {
				l.Debugw("consumer not found on this pod", "found", found, "name", consumerName)
				continue
			}

			// stop consumer
			wg.Add(1)
			stopped++
			consumer := c.(Consumer)

			go ls.stopConsumer(ctx, &wg, consumer)
		}

		wg.Wait()
		l.Debugw("wait completed")

		if stopped > 0 {
			ls.state.Put(ctx, event.Id, state.LogStateFinished)
			l.Infow("execution logs consumers stopped", "id", event.Id, "stopped", stopped)
		} else {
			l.Debugw("no consumers found on this pod to stop")
		}

	}
}

func (ls *LogsService) stopConsumer(ctx context.Context, wg *sync.WaitGroup, consumer Consumer) {
	defer wg.Done()

	var (
		info       *jetstream.ConsumerInfo
		err        error
		l          = ls.log
		retries    = 0
		maxRetries = 50
	)

	l.Debugw("stopping consumer", "name", consumer.Name)

	for {
		info, err = consumer.Instance.Info(ctx)
		if err != nil {
			l.Errorw("error getting consumer info", "error", err, "name", consumer.Name)
			return
		}

		nothingToProcess := info.NumAckPending == 0 && info.NumPending == 0
		messagesDelivered := info.Delivered.Consumer > 0 && info.Delivered.Stream > 0

		l.Debugw("consumer info", "nothingToProcess", nothingToProcess, "messagesDelivered", messagesDelivered, "info", info)

		// check if there was some messages processed
		if nothingToProcess && messagesDelivered {
			consumer.Context.Stop()
			ls.consumerInstances.Delete(consumer.Name)
			l.Infow("stopping and removing consumer", "name", consumer.Name, "consumerSeq", info.Delivered.Consumer, "streamSeq", info.Delivered.Stream, "last", info.Delivered.Last)
			return
		}

		// retry if there is no messages processed as there could be slower logs
		retries++
		if retries >= maxRetries {
			l.Errorw("error stopping consumer", "error", err, "name", consumer.Name, "consumerSeq", info.Delivered.Consumer, "streamSeq", info.Delivered.Stream, "last", info.Delivered.Last)
			return
		}

		// pause a little bit
		l.Debugw("waiting for consumer to finish", "name", consumer.Name, "retries", retries, "consumerSeq", info.Delivered.Consumer, "streamSeq", info.Delivered.Stream, "last", info.Delivered.Last)
		time.Sleep(ls.stopPauseInterval)
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
