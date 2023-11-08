package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	minioconnecter "github.com/kubeshop/testkube/pkg/storage/minio"
)

var _ Consumer = &MinioConsumer{}

// MinioConsumer creates new MinioSubscriber which will send data to local MinIO bucket
func NewMinioConsumer(endpoint, accessKeyID, secretAccessKey, region, token, bucket string, ssl bool) *MinioConsumer {
	c := &MinioConsumer{
		minioConnecter: minioconnecter.NewConnecter(endpoint, accessKeyID, secretAccessKey, region, token, bucket, ssl, log.DefaultLogger),
		Log:            log.DefaultLogger,
		bucket:         bucket,
		region:         region,
		disconnected:   false,
	}

	return c
}

type MinioConsumer struct {
	minioConnecter *minioconnecter.Connecter
	bucket         string
	region         string
	Log            *zap.SugaredLogger
	disconnected   bool
}

func (s *MinioConsumer) Notify(id string, e events.LogChunk) error {
	if s.disconnected {
		s.Log.Debugw("minio consumer disconnected", "id", id)
		return nil
	}
	minioClient, err := s.minioConnecter.GetClient()
	if err != nil {
		return err
	}

	exists, err := minioClient.BucketExists(context.TODO(), s.bucket)
	if err != nil {
		return err
	}

	if !exists {
		err = minioClient.MakeBucket(context.TODO(), s.bucket,
			minio.MakeBucketOptions{Region: s.region})
		if err != nil {
			return err
		}
	}
	file, err := minioClient.GetObject(context.TODO(), s.bucket, id, minio.GetObjectOptions{})
	if err != nil {
		return err
	}

	fileContent, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	chunckToAdd, err := json.Marshal(e)
	if err != nil {
		return err
	}

	fileContent = append(fileContent, chunckToAdd...)

	err = minioClient.RemoveObject(context.TODO(), s.bucket, id, minio.RemoveObjectOptions{ForceDelete: true})
	if err != nil {
		return err
	}

	reader := bytes.NewReader(fileContent)
	_, err = minioClient.PutObject(context.TODO(), s.bucket, id, reader, reader.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})

	if err != nil {
		return err
	}

	return nil
}

func (s *MinioConsumer) Stop(id string) error {
	s.disconnected = true
	s.minioConnecter.Disconnect()
	return nil
}

func (s *MinioConsumer) Name() string {
	return "minio"
}
