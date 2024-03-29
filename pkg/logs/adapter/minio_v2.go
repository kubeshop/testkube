package adapter

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/events"
	minioconnecter "github.com/kubeshop/testkube/pkg/storage/minio"
)

// DefaultDataDir is a default directory where logs are stored (logs-service Dockerfile creates this directory)
const DefaultDataDir = "/data"

var _ Adapter = &MinioV2Adapter{}

// NewMinioV2Adapter creates new MinioV2Adapter which will send data to local MinIO bucket
func NewMinioV2Adapter(endpoint, accessKeyID, secretAccessKey, region, token, bucket string, ssl, skipVerify bool, certFile, keyFile, caFile string) (*MinioV2Adapter, error) {
	ctx := context.Background()
	opts := minioconnecter.GetTLSOptions(ssl, skipVerify, certFile, keyFile, caFile)
	c := &MinioV2Adapter{
		minioConnecter: minioconnecter.NewConnecter(endpoint, accessKeyID, secretAccessKey, region, token, bucket, log.DefaultLogger, opts...),
		log:            log.DefaultLogger,
		bucket:         bucket,
		region:         region,
		files:          make(map[string]*os.File),
		path:           DefaultDataDir,
	}
	minioClient, err := c.minioConnecter.GetClient()
	if err != nil {
		c.log.Errorw("error connecting to minio", "err", err)
		return c, err
	}

	c.minioClient = minioClient
	exists, err := c.minioClient.BucketExists(ctx, c.bucket)
	if err != nil {
		c.log.Errorw("error checking if bucket exists", "err", err)
		return c, err
	}

	if !exists {
		err = c.minioClient.MakeBucket(ctx, c.bucket,
			minio.MakeBucketOptions{Region: c.region})
		if err != nil {
			c.log.Errorw("error creating bucket", "err", err)
			return c, err
		}
	}
	return c, nil
}

type MinioV2Adapter struct {
	minioConnecter *minioconnecter.Connecter
	minioClient    *minio.Client
	bucket         string
	region         string
	log            *zap.SugaredLogger
	traceMessages  bool
	lock           sync.RWMutex
	path           string
	files          map[string]*os.File
}

func (s *MinioV2Adapter) Init(ctx context.Context, id string) error {
	filePath := filepath.Join(s.path, id)

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	s.putFile(id, file)

	return nil
}

func (s *MinioV2Adapter) putFile(id string, f *os.File) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.files[id] = f
}

func (s *MinioV2Adapter) countFiles() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.files)
}

func (s *MinioV2Adapter) getFile(id string) (f *os.File, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	file, ok := s.files[id]
	if !ok {
		return nil, os.ErrNotExist
	}

	return file, nil
}

func (s *MinioV2Adapter) deleteFile(id string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.files, id)
}

func (s *MinioV2Adapter) WithTraceMessages(enabled bool) *MinioV2Adapter {
	s.traceMessages = enabled
	return s
}

func (s *MinioV2Adapter) WithPath(path string) {
	s.path = path
}

func (s *MinioV2Adapter) Notify(ctx context.Context, id string, e events.Log) error {
	if s.traceMessages {
		s.log.Debugw("minio consumer notify", "id", id, "event", e)
	}

	chunk, err := json.Marshal(e)
	if err != nil {
		return err
	}

	file, err := s.getFile(id)
	if err != nil {
		return err
	}

	_, err = file.Write(append(chunk, []byte("\n")...))

	return err
}

func (s *MinioV2Adapter) Stop(ctx context.Context, id string) error {
	log := s.log.With("id", id)

	log.Debugw("stopping minio consumer")

	file, err := s.getFile(id)
	if err != nil {
		return err
	}

	// rewind file to the beginning
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	info, err := s.minioClient.PutObject(ctx, s.bucket, id, file, stat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		log.Errorw("error putting object", "err", err)
		return err
	}

	log.Debugw("put object successfully", "id", id, "s.bucket", s.bucket, "uploadInfo", info)

	// clean memory
	err = file.Close()

	if err != nil {
		return err
	}

	s.deleteFile(id)

	err = os.Remove(filepath.Join(s.path, id))
	if err != nil {
		return err
	}

	log.Debugw("minio consumer stopped, tmp file removed", "filesInUse", s.countFiles())

	return nil
}

func (s *MinioV2Adapter) Name() string {
	return "minio-v2"
}
