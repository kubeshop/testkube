package storage

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"

	"github.com/minio/minio-go/v7"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/storage"
)

var ErrNotAllowed = errors.New("operation not allowed in cloud mode")

type CloudClient struct {
	executor executor.Executor
}

func NewCloudClient(executor executor.Executor) *CloudClient {
	return &CloudClient{executor: executor}
}

func (c *CloudClient) CreateBucket(bucket string) error {
	return ErrNotAllowed
}

func (c *CloudClient) DeleteBucket(bucket string, force bool) error {
	return ErrNotAllowed
}

func (c *CloudClient) ListBuckets() ([]string, error) {
	return nil, ErrNotAllowed
}

func (c *CloudClient) DownloadFileFromBucket(bucket, bucketFolder, file string) (*minio.Object, error) {
	return nil, ErrNotAllowed
}

func (c *CloudClient) UploadFileToBucket(bucket, bucketFolder, filePath string, reader io.Reader, objectSize int64) error {
	return ErrNotAllowed
}

func (c *CloudClient) GetValidBucketName(parentType string, parentName string) string {
	bucketName := fmt.Sprintf("%s-%s", parentType, parentName)
	if len(bucketName) <= 63 {
		return bucketName
	}

	h := fnv.New32a()
	h.Write([]byte(bucketName))

	return fmt.Sprintf("%s-%d", bucketName[:52], h.Sum32())
}

func (c *CloudClient) DeleteFileFromBucket(bucket, bucketFolder, file string) error {
	return ErrNotAllowed
}

func (c *CloudClient) ListFiles(bucketFolder string) ([]testkube.Artifact, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CloudClient) SaveFile(bucketFolder, filePath string) error {
	//TODO implement me
	panic("implement me")
}

func (c *CloudClient) DownloadFile(bucketFolder, file string) (*minio.Object, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CloudClient) UploadFile(bucketFolder string, filePath string, reader io.Reader, objectSize int64) error {
	//TODO implement me
	panic("implement me")
}

func (c *CloudClient) PlaceFiles(bucketFolders []string, prefix string) error {
	//TODO implement me
	panic("implement me")
}

func (c *CloudClient) DeleteFile(bucketFolder, file string) error {
	//TODO implement me
	panic("implement me")
}

var _ storage.Client = &CloudClient{}
