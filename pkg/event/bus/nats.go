package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := nats.Connect("localhost")
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// Use a WaitGroup to wait for 10 messages to arrive
	wg := sync.WaitGroup{}
	wg.Add(10)

	go func() {
		for {
			nc.Publish("updates", []byte("All is Well"))
			time.Sleep(time.Second)
		}

	}()

	nc.Subscribe("updates", func(m *nats.Msg) {
		fmt.Printf("sub: %+v\n", string(m.Data))
	})

	// Create a queue subscription on "updates" with queue name "workers"
	// events -> webhook.testkube.name
	if _, err := nc.QueueSubscribe("updates", "workers", func(m *nats.Msg) {
		fmt.Printf("queue1: %+v\n", string(m.Data))
		wg.Done()
	}); err != nil {
		log.Fatal(err)
	}
	// Create a queue subscription on "updates" with queue name "workers"
	if _, err := nc.QueueSubscribe("updates", "workers", func(m *nats.Msg) {
		fmt.Printf("queue2: %+v\n", string(m.Data))
		wg.Done()
	}); err != nil {
		log.Fatal(err)
	}

	wg.Wait()
}
