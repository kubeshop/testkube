package consumer

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/logs/events"
)

var _ Consumer = &DummyConsumer{}

// NewS3Subscriber creates new DummySubscriber which will send data to local MinIO bucket
func NewDummyConsumer() *DummyConsumer {
	return &DummyConsumer{}
}

type DummyConsumer struct {
	Bucket string
}

func (s *DummyConsumer) Notify(id string, e events.LogChunk) error {
	fmt.Printf("%s %+v\n", id, e)
	return nil
}

func (s *DummyConsumer) Stop(id string) error {
	fmt.Printf("stopping %s \n", id)
	return nil
}

func (s *DummyConsumer) Name() string {
	return "dummy"
}
