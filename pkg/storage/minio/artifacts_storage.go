package minio

import (
	"context"
	"io"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/storage"
)

type ArtifactClient struct {
	client storage.Client
}

// NewMinIOArtifactClient returns new MinIO client
func NewMinIOArtifactClient(client storage.Client) *ArtifactClient {
	return &ArtifactClient{client: client}
}

// ListFiles lists available files in the bucket from the config
func (c *ArtifactClient) ListFiles(ctx context.Context, executionId, testName, testSuiteName, testWorkflowName string) ([]testkube.Artifact, error) {
	return c.client.ListFiles(ctx, executionId)
}

// DownloadFile downloads file from bucket from the config
func (c *ArtifactClient) DownloadFile(ctx context.Context, file, executionId, testName, testSuiteName, testWorkflowName string) (io.Reader, error) {
	return c.client.DownloadFile(ctx, executionId, file)
}

// DownloadArrchive downloads archive from bucket from the config
func (c *ArtifactClient) DownloadArchive(ctx context.Context, executionId string, masks []string) (io.Reader, error) {
	return c.client.DownloadArchive(ctx, executionId, masks)
}

// UploadFile saves a file to be copied into a running execution
func (c *ArtifactClient) UploadFile(ctx context.Context, bucketFolder, filePath string, reader io.Reader, objectSize int64) error {
	return c.client.UploadFile(ctx, bucketFolder, filePath, reader, objectSize)
}

// PlaceFiles saves the content of the buckets to the filesystem
func (c *ArtifactClient) PlaceFiles(ctx context.Context, bucketFolders []string, prefix string) error {
	return c.client.PlaceFiles(ctx, bucketFolders, prefix)
}

func (c *ArtifactClient) GetValidBucketName(parentType string, parentName string) string {
	return c.client.GetValidBucketName(parentType, parentName)
}

var _ storage.ArtifactsStorage = (*ArtifactClient)(nil)
