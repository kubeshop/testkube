package factory

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/cloud"
	cloudscraper "github.com/kubeshop/testkube/pkg/cloud/data/artifact"
	cloudexecutor "github.com/kubeshop/testkube/pkg/cloud/data/executor"
	"github.com/kubeshop/testkube/pkg/envs"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/scraper"
	"github.com/kubeshop/testkube/pkg/filesystem"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"
)

type ExtractorType string
type UploaderType string

const (
	RecursiveFilesystemExtractor ExtractorType = "RecursiveFilesystemExtractor"
	ArchiveFilesystemExtractor   ExtractorType = "ArchiveFilesystemExtractor"
	MinIOUploader                UploaderType  = "MinIOUploader"
	CloudUploader                UploaderType  = "CloudUploader"
)

func GetScraper(ctx context.Context, params envs.Params, extractorType ExtractorType, uploaderType UploaderType) (scraper.Scraper, error) {
	var extractor scraper.Extractor
	switch extractorType {
	case RecursiveFilesystemExtractor:
		extractor = scraper.NewRecursiveFilesystemExtractor(filesystem.NewOSFileSystem())
	case ArchiveFilesystemExtractor:
		extractor = scraper.NewArchiveFilesystemExtractor(filesystem.NewOSFileSystem())
	default:
		return nil, errors.Errorf("unknown extractor type: %s", extractorType)
	}

	var err error
	var loader scraper.Uploader
	var closeF func() error
	switch uploaderType {
	case MinIOUploader:
		loader, err = getMinIOLoader(params)
		if err != nil {
			return nil, errors.Wrap(err, "error creating minio loader")
		}
	case CloudUploader:
		loader, closeF, err = getCloudLoader(ctx, params)
		if err != nil {
			return nil, errors.Wrap(err, "error creating cloud loader")
		}
		defer closeF()
	default:
		return nil, errors.Errorf("unknown uploader type: %s", uploaderType)
	}

	return scraper.NewExtractLoadScraper(extractor, loader), nil
}

func getCloudLoader(ctx context.Context, params envs.Params) (uploader *cloudscraper.CloudUploader, closeF func() error, err error) {
	output.PrintLog(fmt.Sprintf("%s Uploading artifacts using Cloud Uploader", ui.IconCheckMark))

	grpcConn, err := agent.NewGRPCConnection(ctx, params.CloudAPITLSInsecure, params.CloudAPIURL, log.DefaultLogger)
	if err != nil {
		return nil, nil, err
	}
	closeF = func() error {
		return grpcConn.Close()
	}
	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)
	cloudExecutor := cloudexecutor.NewCloudGRPCExecutor(grpcClient, params.CloudAPIKey)
	return cloudscraper.NewCloudUploader(cloudExecutor), closeF, nil
}

func getMinIOLoader(params envs.Params) (*scraper.MinIOUploader, error) {
	output.PrintLog(fmt.Sprintf("%s Uploading artifacts using MinIO Uploader", ui.IconCheckMark))
	return scraper.NewMinIOUploader(
		params.Endpoint,
		params.AccessKeyID,
		params.SecretAccessKey,
		params.Region,
		params.Token,
		params.Bucket,
		params.Ssl,
	)
}
