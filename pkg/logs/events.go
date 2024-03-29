package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/logs/adapter"
	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/logs/state"
)

const (
	StatusPending  byte = 1
	StatusFinished byte = 2

	StreamPrefix = "log"

	StartQueue = "logsstart"
	StopQueue  = "logsstop"

	LogStartSubject = "events.logs.start"
	LogStopSubject  = "events.logs.stop"
)

var (
	StartSubjects = map[string]string{
		"test":    testkube.TestStartSubject,
		"generic": LogStartSubject,
	}

	StopSubjects = map[string]string{
		"test":    testkube.TestStopSubject,
		"generic": LogStopSubject,
	}
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

	err := a.Init(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "can't init adapter")
	}

	return ls.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name: name,
		// Durable: name,
		// FilterSubject: streamName,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
}

// handleMessage will handle incoming message from logs stream and proxy it to given adapter
func (ls *LogsService) handleMessage(ctx context.Context, a adapter.Adapter, id string) func(msg jetstream.Msg) {
	log := ls.log.With("id", id, "adapter", a.Name())

	return func(msg jetstream.Msg) {
		if ls.traceMessages {
			log.Debugw("got message", "data", string(msg.Data()))
		}

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

		err = a.Notify(ctx, id, logChunk)
		if err != nil {
			if err := msg.Nak(); err != nil {
				log.Errorw("error nacking message", "error", err)
				return
			}
			log.Errorw("error notifying adapter", "error", err)
			return
		}

		if err := msg.Ack(); err != nil {
			log.Errorw("error acking message", "error", err)
		}
	}
}

// handleStart will handle start event and create logs consumers, also manage state of given (execution) id
func (ls *LogsService) handleStart(ctx context.Context, subject string) func(msg *nats.Msg) {
	return func(msg *nats.Msg) {
		event := events.Trigger{}
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			ls.log.Errorw("can't handle start event", "error", err)
			return
		}
		id := event.ResourceId
		log := ls.log.With("id", id, "event", "start")

		ls.state.Put(ctx, id, state.LogStatePending)

		s, err := ls.logStream.Init(ctx, id)
		if err != nil {
			ls.log.Errorw("error creating stream", "error", err, "id", id)
			return
		}

		log.Infow("stream created", "stream", s)

		streamName := StreamPrefix + id

		// for each adapter create NATS consumer and consume stream from it e.g. cloud s3 or others
		for i, adapter := range ls.adapters {
			l := log.With("adapter", adapter.Name())
			c, err := ls.initConsumer(ctx, adapter, streamName, id, i)
			if err != nil {
				log.Errorw("error creating consumer", "error", err)
				return
			}

			// handle message per each adapter
			l.Infow("consumer created", "consumer", c.CachedInfo(), "stream", streamName)
			cons, err := c.Consume(ls.handleMessage(ctx, adapter, id))
			if err != nil {
				log.Errorw("error creating consumer", "error", err, "consumer", c.CachedInfo())
				continue
			}

			consumerName := id + "_" + adapter.Name() + "_" + subject
			// store consumer instance so we can stop it later in StopSubject handler
			ls.consumerInstances.Store(consumerName, Consumer{
				Name:     consumerName,
				Context:  cons,
				Instance: c,
			})

			l.Infow("consumer started", "adapter", adapter.Name(), "id", id, "stream", streamName)
		}

		// confirm when reply is set
		if msg.Reply != "" {
			// reply to start event that everything was initialized correctly
			err = msg.Respond([]byte("ok"))
			if err != nil {
				log.Errorw("error responding to start event", "error", err)
				return
			}
		}
	}

}

// handleStop will handle stop event and stop logs consumers, also clean consumers state
func (ls *LogsService) handleStop(ctx context.Context, group string) func(msg *nats.Msg) {
	return func(msg *nats.Msg) {
		var (
			wg      sync.WaitGroup
			stopped = 0
			event   = events.Trigger{}
		)

		if ls.traceMessages {
			ls.log.Debugw("got stop event", "data", string(msg.Data))
		}

		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			ls.log.Errorw("can't handle stop event", "error", err)
			return
		}

		id := event.ResourceId

		l := ls.log.With("id", id, "event", "stop")

		if msg.Reply != "" {
			err = msg.Respond([]byte("stop-queued"))
			if err != nil {
				l.Errorw("error responding to stop event", "error", err)
			}
		}

		for _, adapter := range ls.adapters {
			consumerName := id + "_" + adapter.Name() + "_" + group

			// locate consumer on this pod
			c, found := ls.consumerInstances.Load(consumerName)
			if !found {
				l.Debugw("consumer not found on this pod", "found", found, "name", consumerName)
				continue
			}
			l.Debugw("consumer instance found", "c", c, "found", found, "name", consumerName)

			// stop consumer
			wg.Add(1)
			stopped++
			consumer := c.(Consumer)

			go ls.stopConsumer(ctx, &wg, consumer, adapter, id)
		}

		wg.Wait()
		l.Debugw("wait completed")

		if stopped > 0 {
			ls.state.Put(ctx, event.ResourceId, state.LogStateFinished)
			l.Infow("execution logs consumers stopped", "id", event.ResourceId, "stopped", stopped)
		} else {
			l.Debugw("no consumers found on this pod to stop")
		}
	}
}

func (ls *LogsService) stopConsumer(ctx context.Context, wg *sync.WaitGroup, consumer Consumer, adapter adapter.Adapter, id string) {
	defer wg.Done()

	var (
		info       *jetstream.ConsumerInfo
		err        error
		l          = ls.log
		retries    = 0
		maxRetries = 50
	)

	defer func() {
		// send log finish message as consumer listening for logs needs to be closed
		err = ls.logStream.Finish(ctx, id)
		if err != nil {
			ls.log.Errorw("log stream finish error")
		}
	}()

	l.Debugw("stopping consumer", "name", consumer.Name)

	// stop nats consumer
	defer consumer.Context.Stop()

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

			// delete nats consumer instance from memory
			ls.consumerInstances.Delete(consumer.Name)
			l.Infow("stopping and removing consumer", "name", consumer.Name, "consumerSeq", info.Delivered.Consumer, "streamSeq", info.Delivered.Stream, "last", info.Delivered.Last)

			// call adapter stop to handle given id
			err := adapter.Stop(ctx, id)
			if err != nil {
				l.Errorw("stop error", "adapter", adapter.Name(), "error", err)
			}
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
