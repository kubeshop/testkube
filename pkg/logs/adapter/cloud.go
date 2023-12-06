package adapter

import "github.com/kubeshop/testkube/pkg/logs/events"

var _ Adapter = &CloudAdapter{}

// NewCloudConsumer creates new CloudSubscriber which will send data to local MinIO bucket
func NewCloudConsumer() *CloudAdapter {
	return &CloudAdapter{}
}

type CloudAdapter struct {
	Bucket string
}

func (s *CloudAdapter) Notify(id string, e events.LogChunk) error {
	panic("not implemented")
}

func (s *CloudAdapter) Stop(id string) error {
	panic("not implemented")
}

func (s *CloudAdapter) Name() string {
	return "cloud"
}
