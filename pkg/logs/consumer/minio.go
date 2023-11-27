package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	minioconnecter "github.com/kubeshop/testkube/pkg/storage/minio"
)

const (
	defaultBufferSize = 1024 * 100 // 100KB
	defaultWriteSize  = 1024 * 80  // 80KB
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

	c.minioClient = minioClient
	exists, err := c.minioClient.BucketExists(context.TODO(), c.bucket)
	if err != nil {
		c.Log.Errorw("error checking if bucket exists", "err", err)
		return c
	}

	if !exists {
		err = c.minioClient.MakeBucket(context.TODO(), c.bucket,
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
	minioClient    *minio.Client
	bucket         string
	region         string
	Log            *zap.SugaredLogger
	disconnected   bool
	buffInfos      map[string]BufferInfo
	mapLock        sync.RWMutex
}

func (s *MinioConsumer) Notify(id string, e events.LogChunk) error {
	if s.disconnected {
		s.Log.Debugw("minio consumer disconnected", "id", id)
		return ErrMinioConsumerDisconnected{}
	}

	if _, ok := s.GetBuffInfo(id); !ok {
		s.UpdateBuffInfo(id, BufferInfo{Buffer: bytes.NewBuffer(make([]byte, 0, defaultBufferSize)), Part: 0})
	}

	chunckToAdd, err := json.Marshal(e)
	if err != nil {
		return err
	}
	chunckToAdd = append(chunckToAdd, []byte("\n")...)
	buffInfo, _ := s.GetBuffInfo(id)
	writer := buffInfo.Buffer
	_, err = writer.Write(chunckToAdd)
	if err != nil {
		return err
	}

	if writer.Len() > defaultWriteSize {
		buffInfo.Buffer = bytes.NewBuffer(make([]byte, 0, defaultBufferSize))
		name := id + "-" + strconv.Itoa(buffInfo.Part)
		buffInfo.Part++
		s.UpdateBuffInfo(id, buffInfo)
		go s.putData(name, writer)
	}

	return nil
}

func (s *MinioConsumer) putData(name string, buffer *bytes.Buffer) {
	if buffer != nil && buffer.Len() != 0 {
		_, err := s.minioClient.PutObject(context.TODO(), s.bucket, name, buffer, int64(buffer.Len()), minio.PutObjectOptions{ContentType: "application/octet-stream"})
		if err != nil {
			s.Log.Errorw("error putting object", "err", err)
		}
	} else {
		s.Log.Warn("empty buffer for name: ", name)
	}

}

func (s *MinioConsumer) combineData(minioClient *minio.Client, id string, parts int, deleteIntermediaryData bool) error {
	buffer := bytes.NewBuffer(make([]byte, 0, parts*defaultBufferSize))
	for i := 0; i < parts; i++ {
		objInfo, err := minioClient.GetObject(context.TODO(), s.bucket, fmt.Sprintf("%s-%d", id, i), minio.GetObjectOptions{})
		if err != nil {
			s.Log.Errorw("error getting object", "err", err)
		}
		_, err = buffer.ReadFrom(objInfo)
		if err != nil {
			s.Log.Errorw("error reading object", "err", err)
		}
	}
	_, err := minioClient.PutObject(context.TODO(), s.bucket, id, buffer, int64(buffer.Len()), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		s.Log.Errorw("error putting object", "err", err)
		return err
	}

	if deleteIntermediaryData {
		for i := 0; i < parts; i++ {
			err = minioClient.RemoveObject(context.TODO(), s.bucket, fmt.Sprintf("%s-%d", id, i), minio.RemoveObjectOptions{})
			if err != nil {
				s.Log.Errorw("error removing object", "err", err)
			}
		}
	}
	buffer.Reset()
	return nil
}

func (s *MinioConsumer) Stop(id string) error {
	buffInfo, _ := s.GetBuffInfo(id)
	name := id + "-" + strconv.Itoa(buffInfo.Part)
	s.putData(name, buffInfo.Buffer)
	parts := buffInfo.Part + 1
	s.DeleteBuffInfo(id)
	return s.combineData(s.minioClient, id, parts, true)
}

func (s *MinioConsumer) Name() string {
	return "minio"
}

func (s *MinioConsumer) GetBuffInfo(id string) (BufferInfo, bool) {
	s.mapLock.RLock()
	defer s.mapLock.RUnlock()
	buffInfo, ok := s.buffInfos[id]
	return buffInfo, ok
}

func (s *MinioConsumer) UpdateBuffInfo(id string, buffInfo BufferInfo) {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()
	s.buffInfos[id] = buffInfo
}

func (s *MinioConsumer) DeleteBuffInfo(id string) {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()
	delete(s.buffInfos, id)
}
