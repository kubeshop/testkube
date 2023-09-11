package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
)

func TestLogs(t *testing.T) {

	// connect to nats server
	nc, _ := nats.Connect(nats.DefaultURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("%+v\n", "a1")

	err := nc.Publish("aaaa", []byte("hello message 1"))
	assert.NoError(t, err)

	js, err := jetstream.New(nc)
	assert.NoError(t, err)
	fmt.Printf("%+v\n", "a00000")

	// create a stream (this is an idempotent operation)
	s, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "LOGS",
		Subjects: []string{"LOGS.*"},
	})
	defer js.DeleteStream(ctx, "LOGS")

	assert.NoError(t, err)

	fmt.Printf("%+v\n", "a2")
	fmt.Printf("%+v\n", s.CachedInfo())

	// Create durable consumer
	c, err := s.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "AAA",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	assert.NoError(t, err)
	fmt.Printf("%+v\n", "a3")

	var wg sync.WaitGroup

	fmt.Printf("%+v\n", "AAAA")

	wg.Add(1)
	go func() {
		fmt.Printf("%+v\n", "START")

		// Publish some messages
		for i := 0; i < 100; i++ {
			js.Publish(ctx, "LOGS.a123", []byte("hello message "+strconv.Itoa(i)))
			fmt.Printf("Published hello message %d\n", i)
		}
		for i := 0; i < 100; i++ {
			js.Publish(ctx, "LOGS.b999", []byte("hello message "+strconv.Itoa(i)))
			fmt.Printf("Published hello message %d\n", i)
		}
	}()

	wg.Add(1)
	go func() {
		messageCounter := 0
		// Receive messages continuously in a callback
		cons, _ := c.Consume(func(msg jetstream.Msg) {
			msg.Ack()
			fmt.Printf("Received a JetStream message via callback: %s\n", string(msg.Data()))
			messageCounter++
			fmt.Printf("%+v\n", messageCounter)
		})

		cons.Stop()
	}()

	t.Fail()

}
