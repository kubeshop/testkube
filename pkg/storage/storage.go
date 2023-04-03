package storage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Client is storage client abstraction
type Client interface {
	ClientBucket
	ClientImplicitBucket
}

// ClientImplicitBucket is storage client abstraction where bucket name is provided from config
type ClientImplicitBucket interface {
	ListFiles(ctx context.Context, bucketFolder string) ([]testkube.Artifact, error)
	SaveFile(ctx context.Context, bucketFolder, filePath string) error
	DownloadFile(ctx context.Context, bucketFolder, file string) (*minio.Object, error)
	UploadFile(ctx context.Context, bucketFolder string, filePath string, reader io.Reader, objectSize int64) error
	PlaceFiles(ctx context.Context, bucketFolders []string, prefix string) error
	DeleteFile(ctx context.Context, bucketFolder, file string) error
}

// ClientBucket is storage client abstraction where you have to specify bucket name
type ClientBucket interface {
	CreateBucket(ctx context.Context, bucket string) error
	DeleteBucket(ctx context.Context, bucket string, force bool) error
	ListBuckets(ctx context.Context) ([]string, error)
	DownloadFileFromBucket(ctx context.Context, bucket, bucketFolder, file string) (*minio.Object, error)
	UploadFileToBucket(ctx context.Context, bucket, bucketFolder, filePath string, reader io.Reader, objectSize int64) error
	GetValidBucketName(parentType string, parentName string) string
	DeleteFileFromBucket(ctx context.Context, bucket, bucketFolder, file string) error
}
