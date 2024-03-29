package adapter

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

var _ Adapter = &MinioAdapter{}

type ErrMinioAdapterDisconnected struct {
}

func (e ErrMinioAdapterDisconnected) Error() string {
	return "minio consumer disconnected"
}

type ErrIdNotFound struct {
	Id string
}

func (e ErrIdNotFound) Error() string {
	return fmt.Sprintf("id %s not found", e.Id)
}

type ErrChunckTooBig struct {
	Length int
}

func (e ErrChunckTooBig) Error() string {
	return fmt.Sprintf("chunk too big: %d", e.Length)
}

type BufferInfo struct {
	Buffer *bytes.Buffer
	Part   int
}

// NewMinioAdapter creates new MinioAdapter which will send data to local MinIO bucket
func NewMinioAdapter(endpoint, accessKeyID, secretAccessKey, region, token, bucket string, ssl, skipVerify bool, certFile, keyFile, caFile string) (*MinioAdapter, error) {
	ctx := context.TODO()
	opts := minioconnecter.GetTLSOptions(ssl, skipVerify, certFile, keyFile, caFile)
	c := &MinioAdapter{
		minioConnecter: minioconnecter.NewConnecter(endpoint, accessKeyID, secretAccessKey, region, token, bucket, log.DefaultLogger, opts...),
		Log:            log.DefaultLogger,
		bucket:         bucket,
		region:         region,
		disconnected:   false,
		buffInfos:      make(map[string]BufferInfo),
	}
	minioClient, err := c.minioConnecter.GetClient()
	if err != nil {
		c.Log.Errorw("error connecting to minio", "err", err)
		return c, err
	}

	c.minioClient = minioClient
	exists, err := c.minioClient.BucketExists(ctx, c.bucket)
	if err != nil {
		c.Log.Errorw("error checking if bucket exists", "err", err)
		return c, err
	}

	if !exists {
		err = c.minioClient.MakeBucket(ctx, c.bucket,
			minio.MakeBucketOptions{Region: c.region})
		if err != nil {
			c.Log.Errorw("error creating bucket", "err", err)
			return c, err
		}
	}
	return c, nil
}

type MinioAdapter struct {
	minioConnecter *minioconnecter.Connecter
	minioClient    *minio.Client
	bucket         string
	region         string
	Log            *zap.SugaredLogger
	disconnected   bool
	buffInfos      map[string]BufferInfo
	mapLock        sync.RWMutex
	traceMessages  bool
}

func (s *MinioAdapter) Init(ctx context.Context, id string) error {
	return nil
}

func (s *MinioAdapter) WithTraceMessages(enabled bool) {
	s.traceMessages = enabled
}

func (s *MinioAdapter) Notify(ctx context.Context, id string, e events.Log) error {
	if s.traceMessages {
		s.Log.Debugw("minio consumer notify", "id", id, "event", e)
	}
	if s.disconnected {
		s.Log.Debugw("minio consumer disconnected", "id", id)
		return ErrMinioAdapterDisconnected{}
	}

	buffInfo, ok := s.GetBuffInfo(id)
	if !ok {
		buffInfo = BufferInfo{Buffer: bytes.NewBuffer(make([]byte, 0, defaultBufferSize)), Part: 0}
		s.UpdateBuffInfo(id, buffInfo)
	}

	chunckToAdd, err := json.Marshal(e)
	if err != nil {
		return err
	}

	if len(chunckToAdd) > defaultWriteSize {
		s.Log.Warnw("chunck too big", "length", len(chunckToAdd))
		return ErrChunckTooBig{len(chunckToAdd)}
	}

	chunckToAdd = append(chunckToAdd, []byte("\n")...)

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
		go s.putData(context.TODO(), name, writer)
	}

	return nil
}

func (s *MinioAdapter) putData(ctx context.Context, name string, buffer *bytes.Buffer) {
	if buffer != nil && buffer.Len() != 0 {
		_, err := s.minioClient.PutObject(ctx, s.bucket, name, buffer, int64(buffer.Len()), minio.PutObjectOptions{ContentType: "application/octet-stream"})
		if err != nil {
			s.Log.Errorw("error putting object", "err", err)
		}
		s.Log.Debugw("put object successfully", "name", name, "s.bucket", s.bucket)
	} else {
		s.Log.Warn("empty buffer for name: ", name)
	}

}

func (s *MinioAdapter) combineData(ctxt context.Context, minioClient *minio.Client, id string, parts int, deleteIntermediaryData bool) error {
	var returnedError []error
	returnedError = nil
	buffer := bytes.NewBuffer(make([]byte, 0, parts*defaultBufferSize))
	for i := 0; i < parts; i++ {
		objectName := fmt.Sprintf("%s-%d", id, i)
		if s.objectExists(objectName) {
			objInfo, err := minioClient.GetObject(ctxt, s.bucket, objectName, minio.GetObjectOptions{})
			if err != nil {
				s.Log.Errorw("error getting object", "err", err)
				returnedError = append(returnedError, err)
			}
			_, err = buffer.ReadFrom(objInfo)
			if err != nil {
				s.Log.Errorw("error reading object", "err", err)
				returnedError = append(returnedError, err)
			}
		}
	}

	info, err := minioClient.PutObject(ctxt, s.bucket, id, buffer, int64(buffer.Len()), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		s.Log.Errorw("error putting object", "err", err)
		return err
	}
	s.Log.Debugw("put object successfully", "id", id, "s.bucket", s.bucket, "parts", parts, "uploadInfo", info)

	if deleteIntermediaryData {
		for i := 0; i < parts; i++ {
			objectName := fmt.Sprintf("%s-%d", id, i)
			if s.objectExists(objectName) {
				err = minioClient.RemoveObject(ctxt, s.bucket, objectName, minio.RemoveObjectOptions{})
				if err != nil {
					s.Log.Errorw("error removing object", "err", err)
					returnedError = append(returnedError, err)
				}
			}
		}
	}

	buffer.Reset()
	if len(returnedError) == 0 {
		return nil
	}
	return fmt.Errorf("executed with errors: %v", returnedError)
}

func (s *MinioAdapter) objectExists(objectName string) bool {
	_, err := s.minioClient.StatObject(context.Background(), s.bucket, objectName, minio.StatObjectOptions{})
	return err == nil
}

func (s *MinioAdapter) Stop(ctx context.Context, id string) error {
	s.Log.Debugw("minio consumer stop", "id", id)
	buffInfo, ok := s.GetBuffInfo(id)
	if !ok {
		return ErrIdNotFound{id}
	}
	name := id + "-" + strconv.Itoa(buffInfo.Part)
	s.putData(ctx, name, buffInfo.Buffer)
	parts := buffInfo.Part + 1
	s.DeleteBuffInfo(id)
	return s.combineData(ctx, s.minioClient, id, parts, true)
}

func (s *MinioAdapter) Name() string {
	return "minio"
}

func (s *MinioAdapter) GetBuffInfo(id string) (BufferInfo, bool) {
	s.mapLock.RLock()
	defer s.mapLock.RUnlock()
	buffInfo, ok := s.buffInfos[id]
	return buffInfo, ok
}

func (s *MinioAdapter) UpdateBuffInfo(id string, buffInfo BufferInfo) {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()
	s.buffInfos[id] = buffInfo
	s.Log.Debugw("minioAdapter: updated buff info", "id", id, "bufInfosCount", len(s.buffInfos))
}

func (s *MinioAdapter) DeleteBuffInfo(id string) {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()
	delete(s.buffInfos, id)
	s.Log.Debugw("minioAdapter: deleted buff info", "id", id, "bufInfosCount", len(s.buffInfos))
}
