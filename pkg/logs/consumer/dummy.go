package consumer

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/logs/events"
)

var _ Adapter = &DummyAdapter{}

// NewS3Subscriber creates new DummySubscriber which will send data to local MinIO bucket
func NewDummyAdapter() *DummyAdapter {
	return &DummyAdapter{}
}

type DummyAdapter struct {
	Bucket string
}

func (s *DummyAdapter) Notify(id string, e events.LogChunk) error {
	fmt.Printf("%s %+v\n", id, e)
	return nil
}

func (s *DummyAdapter) Stop(id string) error {
	fmt.Printf("stopping %s \n", id)
	return nil
}

func (s *DummyAdapter) Name() string {
	return "dummy"
}
