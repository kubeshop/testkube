package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MaxConcurrentDownloads is the maximum number of concurrent file downloads
// allowed when building artifact archives.
const MaxConcurrentDownloads = 10

// Client is storage client abstraction
//
//go:generate go tool mockgen -destination=./storage_mock.go -package=storage "github.com/kubeshop/testkube/pkg/storage" Client
type Client interface {
	ClientBucket
	ClientImplicitBucket
}

// ClientImplicitBucket is storage client abstraction where bucket name is provided from config
type ClientImplicitBucket interface {
	IsConnectionPossible(ctx context.Context) (bool, error)
	ListFiles(ctx context.Context, bucketFolder string) ([]testkube.Artifact, error)
	SaveFile(ctx context.Context, bucketFolder, filePath string) error
	DownloadFile(ctx context.Context, bucketFolder, file string) (*minio.Object, error)
	DownloadArchive(ctx context.Context, bucketFolder string, masks []string) (io.Reader, error)
	UploadFile(ctx context.Context, bucketFolder string, filePath string, reader io.Reader, objectSize int64) error
	PlaceFiles(ctx context.Context, bucketFolders []string, prefix string) error
	DeleteFile(ctx context.Context, bucketFolder, file string) error
}

// ObjectInfo is an object in a bucket, carrying the metadata needed to choose between
// several that share a prefix.
//
// testkube.Artifact cannot serve here: it has no modification time, so there would be
// no way to prefer the most recent of several matches, and its Size is an int32, which
// silently truncates above 2 GiB - an ordinary size for a dependency cache.
type ObjectInfo struct {
	// Key is the full object name, not relative to the queried prefix, because the
	// caller needs it back to address the object.
	Key          string
	Size         int64
	LastModified time.Time
}

// ClientBucket is storage client abstraction where you have to specify bucket name
type ClientBucket interface {
	CreateBucket(ctx context.Context, bucket string) error
	// ListObjectsFromBucket lists objects under a prefix, stopping after limit entries.
	// limit of 0 means unlimited.
	ListObjectsFromBucket(ctx context.Context, bucket, prefix string, limit int) ([]ObjectInfo, error)
	DeleteBucket(ctx context.Context, bucket string, force bool) error
	BucketExists(ctx context.Context, bucket string) (bool, error)
	ListBuckets(ctx context.Context) ([]string, error)
	ListFilesFromBucket(ctx context.Context, bucket, bucketFolder string) ([]testkube.Artifact, error)
	DownloadFileFromBucket(ctx context.Context, bucket, bucketFolder, file string) (io.Reader, minio.ObjectInfo, error)
	DownloadArchiveFromBucket(ctx context.Context, bucket, bucketFolder string, masks []string) (io.Reader, error)
	UploadFileToBucket(ctx context.Context, bucket, bucketFolder, filePath string, reader io.Reader, objectSize int64) error
	GetValidBucketName(parentType string, parentName string) string
	DeleteFileFromBucket(ctx context.Context, bucket, bucketFolder, file string) error
	PresignDownloadFileFromBucket(ctx context.Context, bucket, bucketFolder, file string, expires time.Duration) (string, error)
	PresignUploadFileToBucket(ctx context.Context, bucket, bucketFolder, filePath string, expires time.Duration) (string, error)
}
