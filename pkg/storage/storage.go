package storage

import (
	"io"

	"github.com/minio/minio-go/v7"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Client is storage client abstraction
type Client interface {
	ClientBucket
	ClientImplicitBucket
}

// ClientImplicitBucket is storage client abstraction where bucket name is providet from config
type ClientImplicitBucket interface {
	ListFiles(bucketFolder string) ([]testkube.Artifact, error)
	SaveFile(bucketFolder, filePath string) error
	DownloadFile(bucketFolder, file string) (*minio.Object, error)
	UploadFile(bucketFolder string, filePath string, reader io.Reader, objectSize int64) error
	PlaceFiles(bucketFolders []string, prefix string) error
	DeleteFile(bucketFolder, file string) error
}

// ClientBucket is storage client abstraction where you have to specify bucket name
type ClientBucket interface {
	CreateBucket(bucket string) error
	DeleteBucket(bucket string, force bool) error
	ListBuckets() ([]string, error)
	ListFilesFromBucket(bucket string) ([]testkube.Artifact, error)
	SaveFileToBucket(bucket, bucketFolder, filePath string) error
	DownloadFileFromBucket(bucket, bucketFolder, file string) (*minio.Object, error)
	UploadFileToBucket(bucket, bucketFolder, filePath string, reader io.Reader, objectSize int64) error
	GetValidBucketName(parentType string, parentName string) string
	DeleteFileFromBucket(bucket, bucketFolder, file string) error
}
