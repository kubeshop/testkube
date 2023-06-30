package factory

import (
	"context"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
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

func TryGetScrapper(ctx context.Context, params envs.Params) (scraper.Scraper, error) {
	if params.ScrapperEnabled {
		uploader := MinIOUploader
		if params.CloudMode {
			uploader = CloudUploader
		}
		s, err := GetScraper(ctx, params, ArchiveFilesystemExtractor, uploader)
		if err != nil {
			return nil, errors.Wrap(err, "error creating scraper")
		}
		return s, nil
	}

	return nil, nil
}

func GetScraper(ctx context.Context, params envs.Params, extractorType ExtractorType, uploaderType UploaderType) (scraper.Scraper, error) {
	var extractor scraper.Extractor
	switch extractorType {
	case RecursiveFilesystemExtractor:
		extractor = scraper.NewRecursiveFilesystemExtractor(filesystem.NewOSFileSystem())
	case ArchiveFilesystemExtractor:
		var opts []scraper.ArchiveFilesystemExtractorOpts
		if params.CloudMode {
			opts = append(opts, scraper.GenerateTarballMetaFile())
		}
		extractor = scraper.NewArchiveFilesystemExtractor(filesystem.NewOSFileSystem(), opts...)
	default:
		return nil, errors.Errorf("unknown extractor type: %s", extractorType)
	}

	var err error
	var loader scraper.Uploader
	switch uploaderType {
	case MinIOUploader:
		loader, err = getMinIOLoader(params)
		if err != nil {
			return nil, errors.Wrap(err, "error creating minio loader")
		}
	case CloudUploader:
		loader, err = getCloudLoader(ctx, params)
		if err != nil {
			return nil, errors.Wrap(err, "error creating cloud loader")
		}
	default:
		return nil, errors.Errorf("unknown uploader type: %s", uploaderType)
	}

	var cdeventsClient cloudevents.Client
	if params.CDEventsTarget != "" {
		cdeventsClient, err = cloudevents.NewClientHTTP(cloudevents.WithTarget(params.CDEventsTarget))
		if err != nil {
			log.DefaultLogger.Warnf("failed to create cloud event client %w", err)
		}
	}

	return scraper.NewExtractLoadScraper(extractor, loader, cdeventsClient, params.ClusterID, params.DashboardURI), nil
}

func getCloudLoader(ctx context.Context, params envs.Params) (uploader *cloudscraper.CloudUploader, err error) {
	// timeout blocking connection to cloud
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(params.CloudConnectionTimeoutSec)*time.Second)
	defer cancel()

	output.PrintLogf("%s Uploading artifacts using Cloud Uploader (timeout:%ds)", ui.IconCheckMark, params.CloudConnectionTimeoutSec)
	grpcConn, err := agent.NewGRPCConnection(ctxTimeout, params.CloudAPITLSInsecure, params.CloudAPIURL, log.DefaultLogger)
	if err != nil {
		return nil, err
	}
	output.PrintLogf("%s Connected to Testkube Cloud", ui.IconCheckMark)

	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)
	cloudExecutor := cloudexecutor.NewCloudGRPCExecutor(grpcClient, grpcConn, params.CloudAPIKey)
	return cloudscraper.NewCloudUploader(cloudExecutor), nil
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
