package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	const (
		StreamName = "LOGS"
	)
	// In the `jetstream` package, almost all API calls rely on `context.Context` for timeout/cancellation handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nc, _ := nats.Connect(nats.DefaultURL)

	// Create a JetStream management interface
	js, _ := jetstream.New(nc)

	switch os.Args[1] {
	case "delete":
		err := js.DeleteConsumer(ctx, StreamName, "OC")
		fmt.Printf("%+v\n", err)

		err = js.DeleteStream(ctx, StreamName)
		fmt.Printf("%+v\n", err)

	case "create":
		// Create a stream
		s, err := js.CreateStream(ctx, jetstream.StreamConfig{
			Name:     StreamName,
			Subjects: []string{"ORDERS.*"},
			MaxAge:   time.Minute,
			Storage:  jetstream.FileStorage,
		})

		fmt.Printf("%+v\n", err)
		printStreamState(ctx, s, StreamName)

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
			js.Publish(ctx, "ORDERS.a1", []byte("hello message "+strconv.Itoa(i)))
			fmt.Printf("Published hello message %d\n", i)
		}

		// Publish some messages
		for i := 1; i <= 100; i++ {
			js.Publish(ctx, "ORDERS.b2", []byte("hello message "+strconv.Itoa(i)))
			fmt.Printf("Published hello message %d\n", i)
		}

	case "consume":

		c, err := js.Consumer(ctx, StreamName, "OC")
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

		c, err := js.CreateOrUpdateConsumer(ctx, StreamName, jetstream.ConsumerConfig{
			Name:          "AAA1",
			Durable:       "AAA1",
			DeliverPolicy: jetstream.DeliverByStartSequencePolicy,
			OptStartSeq:   uint64(from),
		})

		if err != nil {
			fmt.Printf("%+v\n", err)
			return
		}
		defer js.DeleteConsumer(ctx, StreamName, "AAA1")

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

func printStreamState(ctx context.Context, js jetstream.Stream, name string) {
	info, _ := js.Info(ctx)
	b, _ := json.MarshalIndent(info.State, "", " ")
	fmt.Println("inspecting stream info")
	fmt.Println(string(b))
}
