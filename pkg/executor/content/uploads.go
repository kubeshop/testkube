package content

import (
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
)

// CopyFilesPlacer takes care of downloading the file into the execution
type CopyFilesPlacer struct {
	client storage.Client
}

// NewCopyFilesPlaces creates a new
func NewCopyFilesPlacer(endpoint, accessKeyID, secretAccessKey, location, token string, ssl bool) *CopyFilesPlacer {
	c := minio.NewClient(endpoint, accessKeyID, secretAccessKey, location, token, ssl)
	return &CopyFilesPlacer{
		client: c,
	}
}

// PlaceFiles downloads the files from minio and places them into the /data/uploads directory
func (p CopyFilesPlacer) PlaceFiles(testName, executionBucket string) error {
	prefix := "/data/uploads/"

	buckets := []string{}
	if testName != "" {
		buckets = append(buckets, p.client.GetValidBucketName("test", testName))
	}
	if executionBucket != "" {
		buckets = append(buckets, executionBucket)
	}

	return p.client.PlaceFiles(buckets, prefix)
}
