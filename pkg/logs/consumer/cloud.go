package consumer

import "github.com/kubeshop/testkube/pkg/logs/events"

var _ Adapter = &CloudSubscriber{}

// NewCloudConsumer creates new CloudSubscriber which will send data to local MinIO bucket
func NewCloudConsumer() *CloudSubscriber {
	return &CloudSubscriber{}
}

type CloudSubscriber struct {
	Bucket string
}

func (s *CloudSubscriber) Notify(id string, e events.LogChunk) error {
	panic("not implemented")
}

func (s *CloudSubscriber) Stop(id string) error {
	panic("not implemented")
}

func (s *CloudSubscriber) Name() string {
	return "cloud"
}
