package content

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/ui"
)

// CopyFilesPlacer takes care of downloading the file into the execution
type CopyFilesPlacer struct {
	client storage.Client
}

// NewCopyFilesPlaces creates a new
func NewCopyFilesPlacer(endpoint, accessKeyID, secretAccessKey, location, region, token, bucket string, ssl bool) *CopyFilesPlacer {
	c := minio.NewClient(endpoint, accessKeyID, secretAccessKey, location, region, token, bucket, ssl)
	return &CopyFilesPlacer{
		client: c,
	}
}

// PlaceFiles downloads the files from minio and places them into the /data/uploads directory
// A warning will be shown in case there was an error placing the files.
func (p CopyFilesPlacer) PlaceFiles(testName, executionBucket string) {
	prefix := "/data/uploads/"
	output.PrintEvent(fmt.Sprintf("%s Placing files from buckets into %s", ui.IconFile, prefix))

	buckets := []string{}
	if testName != "" {
		buckets = append(buckets, p.client.GetValidBucketName("test", testName))
	}
	if executionBucket != "" {
		buckets = append(buckets, p.client.GetValidBucketName("execution", executionBucket))
	}

	err := p.client.PlaceFiles(buckets, prefix)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Could not place files: %s", ui.IconWarning, err.Error()))
	}
}
