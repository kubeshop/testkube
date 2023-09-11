package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	// In the `jetstream` package, almost all API calls rely on `context.Context` for timeout/cancellation handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nc, _ := nats.Connect(nats.DefaultURL)

	// Create a JetStream management interface
	js, _ := jetstream.New(nc)

	switch os.Args[1] {
	case "create":
		// Create a stream
		s, _ := js.CreateStream(ctx, jetstream.StreamConfig{
			Name:     "ORDERS",
			Subjects: []string{"ORDERS.*"},
		})
		// // Create durable consumer
		c, err := s.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
			Name:      "OC",
			Durable:   "OC",
			AckPolicy: jetstream.AckExplicitPolicy,
		})

		fmt.Printf("%+v\n", err)
		fmt.Printf("%+v\n", c)

	case "publish":
		// Publish some messages
		for i := 1; i <= 100; i++ {
			js.Publish(ctx, "ORDERS.new", []byte("hello message "+strconv.Itoa(i)))
			fmt.Printf("Published hello message %d\n", i)
		}

		// Publish some messages
		for i := 1; i <= 100; i++ {
			js.Publish(ctx, "ORDERS.old", []byte("hello message "+strconv.Itoa(i)))
			fmt.Printf("Published hello message %d\n", i)
		}

	case "consume":

		c, err := js.Consumer(ctx, "ORDERS", "OC")
		if err != nil {
			fmt.Printf("%+v\n", err)
			return
		}

		// Receive messages continuously in a callback
		var messageCounter int
		cons, _ := c.Consume(func(msg jetstream.Msg) {
			err := msg.Ack()
			if err != nil {
				fmt.Printf("ack error: %+v\n", err)
				return
			}
			messageCounter++
			fmt.Printf("Received %d message via callback: %s\n", messageCounter, string(msg.Data()))
		})
		defer cons.Stop()
		fmt.Printf("%+v\n", messageCounter)

		time.Sleep(time.Hour)

	case "iter":
		from := 0
		if len(os.Args) == 3 {
			from, _ = strconv.Atoi(os.Args[2])
		}

		c, err := js.CreateOrUpdateConsumer(ctx, "ORDERS", jetstream.ConsumerConfig{
			Name:          "AAA1",
			Durable:       "AAA1",
			DeliverPolicy: jetstream.DeliverByStartSequencePolicy,
			OptStartSeq:   uint64(from),
		})

		if err != nil {
			fmt.Printf("%+v\n", err)
			return
		}
		defer js.DeleteConsumer(ctx, "ORDERS", "AAA1")

		messageCounter := 0
		// Iterate over messages continuously
		it, _ := c.Messages()
		for i := 0; i < 10; i++ {
			msg, _ := it.Next()
			msg.Ack()
			fmt.Printf("Received a JetStream message via iterator: %s\n", string(msg.Data()))
			messageCounter++
		}
		it.Stop()

	}

}
