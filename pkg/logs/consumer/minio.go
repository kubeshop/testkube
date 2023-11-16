package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	minioconnecter "github.com/kubeshop/testkube/pkg/storage/minio"
)

const (
	defaultBufferSize = 100 * 1024 * 1024 // 100MB
	defaultWriteSize  = 95 * 1024 * 1024  // 95MB
)

var _ Consumer = &MinioConsumer{}

type ErrMinioConsumerDisconnected struct {
}

func (e ErrMinioConsumerDisconnected) Error() string {
	return "minio consumer disconnected"
}

type BufferInfo struct {
	Buffer *bytes.Buffer
	Part   int
}

// MinioConsumer creates new MinioSubscriber which will send data to local MinIO bucket
func NewMinioConsumer(endpoint, accessKeyID, secretAccessKey, region, token, bucket string, ssl bool) *MinioConsumer {
	c := &MinioConsumer{
		minioConnecter: minioconnecter.NewConnecter(endpoint, accessKeyID, secretAccessKey, region, token, bucket, ssl, log.DefaultLogger),
		Log:            log.DefaultLogger,
		bucket:         bucket,
		region:         region,
		disconnected:   false,
		buffInfos:      make(map[string]BufferInfo),
	}
	minioClient, err := c.minioConnecter.GetClient()
	if err != nil {
		c.Log.Errorw("error connecting to minio", "err", err)
		return c
	}

	exists, err := minioClient.BucketExists(context.TODO(), c.bucket)
	if err != nil {
		c.Log.Errorw("error checking if bucket exists", "err", err)
		return c
	}

	if !exists {
		err = minioClient.MakeBucket(context.TODO(), s.bucket,
			minio.MakeBucketOptions{Region: c.region})
		if err != nil {
			c.Log.Errorw("error creating bucket", "err", err)
			return c
		}
	}
	return c
}

type MinioConsumer struct {
	minioConnecter *minioconnecter.Connecter
	bucket         string
	region         string
	Log            *zap.SugaredLogger
	disconnected   bool
	buffInfos      map[string]BufferInfo
}

func (s *MinioConsumer) Notify(id string, e events.LogChunk) error {
	if s.disconnected {
		s.Log.Debugw("minio consumer disconnected", "id", id)
		return ErrMinioConsumerDisconnected{}
	}

	if _, ok := s.buffInfos[id]; !ok {
		buffInfo := s.buffInfos[id]
		buffInfo.Buffer = bytes.NewBuffer(make([]byte, 0, defaultBufferSize))
	}

	chunckToAdd, err := json.Marshal(e)
	if err != nil {
		return err
	}

	writer := s.buffInfos[id].Buffer
	_, err = writer.Write(chunckToAdd)
	if err != nil {
		return err
	}

	if writer.Len() > defaultWriteSize {

		buffInfo := s.buffInfos[id]
		buffInfo.Buffer = bytes.NewBuffer(make([]byte, 0, defaultBufferSize))
		minioClient, err := s.minioConnecter.GetClient()
		if err != nil {
			return err
		}
		name := id + "-" + strconv.Itoa(s.buffInfos[id].Part)
		buffInfo.Part++
		go s.putData(minioClient, name, writer)
	}

	return nil
}

func (s *MinioConsumer) putData(minioClient *minio.Client, name string, writer *bytes.Buffer) {
	_, err := minioClient.PutObject(context.TODO(), s.bucket, name, writer, int64(writer.Len()), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		s.Log.Errorw("error putting object", "err", err)
	}

}

func (s *MinioConsumer) Stop(id string) error {

	minioClient, err := s.minioConnecter.GetClient()
	if err != nil {
		return err
	}
	name := id + "-" + strconv.Itoa(s.buffInfos[id].Part)
	s.putData(minioClient, name, s.buffInfos[id].Buffer)
	delete(s.buffInfos, id)
	return nil
}

func (s *MinioConsumer) Name() string {
	return "minio"
}
