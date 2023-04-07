package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"

	"google.golang.org/grpc"

	"github.com/kubeshop/testkube/pkg/cloud"

	"github.com/minio/minio-go/v7"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/storage"
)

var ErrNotAllowed = errors.New("operation not allowed in cloud mode")

type CloudClient struct {
	executor executor.Executor
}

func NewCloudClient(cloudClient cloud.TestKubeCloudAPIClient, grpcConn *grpc.ClientConn, apiKey string) *CloudClient {
	return &CloudClient{executor: executor.NewCloudGRPCExecutor(cloudClient, grpcConn, apiKey)}
}

func (c *CloudClient) CreateBucket(ctx context.Context, bucket string) error {
	return ErrNotAllowed
}

func (c *CloudClient) DeleteBucket(ctx context.Context, bucket string, force bool) error {
	return ErrNotAllowed
}

func (c *CloudClient) ListBuckets(ctx context.Context) ([]string, error) {
	return nil, ErrNotAllowed
}

func (c *CloudClient) DownloadFileFromBucket(ctx context.Context, bucket, bucketFolder, file string) (*minio.Object, error) {
	return nil, ErrNotAllowed
}

func (c *CloudClient) UploadFileToBucket(ctx context.Context, bucket, bucketFolder, filePath string, reader io.Reader, objectSize int64) error {
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

func (c *CloudClient) DeleteFileFromBucket(ctx context.Context, bucket, bucketFolder, file string) error {
	return ErrNotAllowed
}

func (c *CloudClient) ListFiles(ctx context.Context, bucketFolder string) ([]testkube.Artifact, error) {
	req := ListFilesRequest{BucketFolder: bucketFolder}
	response, err := c.executor.Execute(ctx, CmdStorageListFiles, req)
	if err != nil {
		return nil, err
	}
	var commandResponse ListFilesResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return nil, err
	}
	return commandResponse.Artifacts, nil
}

func (c *CloudClient) SaveFile(ctx context.Context, bucketFolder, filePath string) error {
	req := SaveFileRequest{
		BucketFolder: "",
		FilePath:     "",
		Reader:       nil,
		ObjectSize:   0,
	}
	response, err := c.executor.Execute(ctx, CmdStorageListFiles, req)
	if err != nil {
		return err
	}
	var commandResponse SaveFileResponse
	if err := json.Unmarshal(response, &commandResponse); err != nil {
		return err
	}
	return nil
}

func (c *CloudClient) DownloadFile(ctx context.Context, bucketFolder, file string) (*minio.Object, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CloudClient) UploadFile(ctx context.Context, bucketFolder string, filePath string, reader io.Reader, objectSize int64) error {
	//TODO implement me
	panic("implement me")
}

func (c *CloudClient) PlaceFiles(ctx context.Context, bucketFolders []string, prefix string) error {
	//TODO implement me
	panic("implement me")
}

func (c *CloudClient) DeleteFile(ctx context.Context, bucketFolder, file string) error {
	//TODO implement me
	panic("implement me")
}

var _ storage.Client = (*CloudClient)(nil)
