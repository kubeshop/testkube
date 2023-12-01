package consumer

import "github.com/kubeshop/testkube/pkg/logs/events"

var _ Adapter = &S3Consumer{}

// NewS3Consumer creates new S3Subscriber which will send data to local MinIO bucket
func NewS3Consumer() *S3Consumer {
	return &S3Consumer{}
}

type S3Consumer struct {
	Bucket string
}

func (s *S3Consumer) Notify(id string, e events.LogChunk) error {
	panic("not implemented")
}

func (s *S3Consumer) Stop(id string) error {
	panic("not implemented")
}

func (s *S3Consumer) Name() string {
	return "s3"
}
