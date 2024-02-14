package storage

import (
	"context"
	"io"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

//go:generate mockgen -destination=./artifacts_mock.go -package=storage "github.com/kubeshop/testkube/pkg/storage" ArtifactsStorage
type ArtifactsStorage interface {
	// ListFiles lists available files in the configured bucket
	ListFiles(ctx context.Context, executionId, testName, testSuiteName, testWorkflowName string) ([]testkube.Artifact, error)
	// DownloadFile downloads file from configured
	DownloadFile(ctx context.Context, file, executionId, testName, testSuiteName, testWorkflowName string) (io.Reader, error)
	// DownloadArchive downloads archive from configured
	DownloadArchive(ctx context.Context, executionId string, masks []string) (io.Reader, error)
	// UploadFile uploads file to configured bucket
	UploadFile(ctx context.Context, bucketFolder string, filePath string, reader io.Reader, objectSize int64) error
	// PlaceFiles saves the content of the bucket folders to the filesystem
	PlaceFiles(ctx context.Context, bucketFolders []string, prefix string) error
	// GetValidBucketName returns a valid bucket name for the given parent type and name
	GetValidBucketName(parentType string, parentName string) string
}
