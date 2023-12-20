package adapter

import "github.com/kubeshop/testkube/pkg/logs/events"

var _ Adapter = &S3Adapter{}

// NewS3Adapter creates new S3Subscriber which will send data to local MinIO bucket
func NewS3Adapter() *S3Adapter {
	return &S3Adapter{}
}

type S3Adapter struct {
	Bucket string
}

func (s *S3Adapter) Notify(id string, e events.Log) error {
	panic("not implemented")
}

func (s *S3Adapter) Stop(id string) error {
	panic("not implemented")
}

func (s *S3Adapter) Name() string {
	return "s3"
}
