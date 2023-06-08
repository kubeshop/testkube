package content

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/storage"
	"github.com/kubeshop/testkube/pkg/ui"
)

// CopyFilesPlacer takes care of downloading the file into the execution
type CopyFilesPlacer struct {
	client storage.Client
}

const (
	UploadsFolder = "/data/uploads/"
)

func NewCopyFilesPlacer(client storage.Client) *CopyFilesPlacer {
	return &CopyFilesPlacer{
		client: client,
	}
}

// PlaceFiles downloads the files from minio and places them into the /data/uploads directory.
// A warning will be shown in case there was an error placing the files.
func (p CopyFilesPlacer) PlaceFiles(ctx context.Context, testName, executionBucket string) {
	output.PrintEvent(fmt.Sprintf("%s Placing files from buckets into %s", ui.IconFile, UploadsFolder))

	var buckets []string
	if testName != "" {
		buckets = append(buckets, p.client.GetValidBucketName("test", testName))
	}
	if executionBucket != "" {
		buckets = append(buckets, p.client.GetValidBucketName("execution", executionBucket))
	}

	err := p.client.PlaceFiles(ctx, buckets, UploadsFolder)
	if err != nil {
		output.PrintLogf("%s Could not place files: %s", ui.IconWarning, err.Error())
	}
}
