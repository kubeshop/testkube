package logs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamName   = "LOGS"
	ConsumerName = "LOGSCONSUMER"
)

func NewLogsCache(js jetstream.JetStream) LogsCache {
	return LogsCache{js: js}
}

type LogsCache struct {
	js jetstream.JetStream
}

func (l LogsCache) Init(ctx context.Context) error {
	// Create a stream
	s, err := l.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     StreamName,
		Subjects: []string{StreamName + ".*"},
		// MaxAge:   time.Minute,
		Storage: jetstream.FileStorage, // durable stream
	})

	printStreamState(ctx, s)

	return err
}

func (l LogsCache) Publish(ctx context.Context, executionId string, line []byte) error {
	// INFO: first result var is publisher ACK can be used for only-once delivery (double ACK pattern)
	_, err := l.js.Publish(ctx, StreamName+"."+executionId, line)
	return err
}

func (l LogsCache) GetRange(ctx context.Context, executionId string, from, count int) (chan []byte, error) {
	ch := make(chan []byte)

	c, err := l.js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Name:          "lc" + executionId,
		Durable:       "lc" + executionId,
		FilterSubject: StreamName + "." + executionId,
		DeliverPolicy: jetstream.DeliverByStartSequencePolicy,
		OptStartSeq:   uint64(from),
	})
	if err != nil {
		return ch, err
	}

	i, err := c.Info(ctx)
	fmt.Printf("%+v\n", i)

	if err != nil {
		return ch, err
	}
	// defer l.js.DeleteConsumer(ctx, StreamName, "AAA1")

	// Iterate over messages continuously
	it, err := c.Messages()
	fmt.Printf("Message() error: %+v\n", err)

	go func() {
		defer it.Stop()
		for i := 0; i < count; i++ {
			msg, err := it.Next()
			if err == jetstream.ErrMsgIteratorClosed {
				fmt.Printf("%+v\n", "no more messages")
				return
			} else if err != nil {
				fmt.Printf("it.Next() error: %+v\n", err)
				return
			}
			msg.Ack()
			ch <- msg.Data()
		}
	}()
	return ch, nil
}

func (l LogsCache) Listen(ctx context.Context, executionId string) (chan []byte, error) {

	ch := make(chan []byte)

	c, err := l.js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
		Name:          "lca" + executionId,
		Durable:       "lca" + executionId,
		FilterSubject: StreamName + "." + executionId,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return ch, err
	}

	cons, err := c.Consume(func(msg jetstream.Msg) {
		err := msg.Ack()
		if err != nil {
			fmt.Printf("ack error: %+v\n", err)
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

func printStreamState(ctx context.Context, js jetstream.Stream) {
	info, _ := js.Info(ctx)
	b, _ := json.MarshalIndent(info.State, "", " ")
	fmt.Println("inspecting stream info")
	fmt.Println(string(b))
}
