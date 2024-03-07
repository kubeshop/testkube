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
		if params.ProMode {
			uploader = CloudUploader
		}
		extractor := RecursiveFilesystemExtractor
		if params.CompressArtifacts {
			extractor = ArchiveFilesystemExtractor
		}

		s, err := GetScraper(ctx, params, extractor, uploader)
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
		if params.ProMode {
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
		loader, err = getMinIOUploader(params)
		if err != nil {
			return nil, errors.Wrap(err, "error creating minio uploader")
		}
	case CloudUploader:
		loader, err = getRemoteStorageUploader(ctx, params)
		if err != nil {
			return nil, errors.Wrap(err, "error creating remote storage uploader")
		}
	default:
		return nil, errors.Errorf("unknown uploader type: %s", uploaderType)
	}

	var cdeventsClient cloudevents.Client
	if params.CDEventsTarget != "" {
		cdeventsClient, err = cloudevents.NewClientHTTP(cloudevents.WithTarget(params.CDEventsTarget))
		if err != nil {
			log.DefaultLogger.Warnf("failed to create cloud event client: %v", err)
		}
	}

	return scraper.NewExtractLoadScraper(extractor, loader, cdeventsClient, params.ClusterID, params.DashboardURI), nil
}

func getRemoteStorageUploader(ctx context.Context, params envs.Params) (uploader *cloudscraper.CloudUploader, err error) {
	// timeout blocking connection to cloud
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(params.ProConnectionTimeoutSec)*time.Second)
	defer cancel()

	output.PrintLogf(
		"%s Uploading artifacts using Remote Storage Uploader (timeout:%ds, agentInsecure:%v, agentSkipVerify: %v, url: %s, scraperSkipVerify: %v)",
		ui.IconCheckMark, params.ProConnectionTimeoutSec, params.ProAPITLSInsecure, params.ProAPISkipVerify, params.ProAPIURL, params.SkipVerify)
	grpcConn, err := agent.NewGRPCConnection(
		ctxTimeout,
		params.ProAPITLSInsecure,
		params.ProAPISkipVerify,
		params.ProAPIURL,
		params.ProAPICertFile,
		params.ProAPIKeyFile,
		params.ProAPICAFile,
		log.DefaultLogger,
	)
	if err != nil {
		return nil, err
	}
	output.PrintLogf("%s Connected to Agent API", ui.IconCheckMark)

	grpcClient := cloud.NewTestKubeCloudAPIClient(grpcConn)
	cloudExecutor := cloudexecutor.NewCloudGRPCExecutor(grpcClient, grpcConn, params.ProAPIKey)
	return cloudscraper.NewCloudUploader(cloudExecutor, params.SkipVerify), nil
}

func getMinIOUploader(params envs.Params) (*scraper.MinIOUploader, error) {
	output.PrintLog(fmt.Sprintf("%s Uploading artifacts using MinIO Uploader", ui.IconCheckMark))
	return scraper.NewMinIOUploader(
		params.Endpoint,
		params.AccessKeyID,
		params.SecretAccessKey,
		params.Region,
		params.Token,
		params.Bucket,
		params.Ssl,
		params.SkipVerify,
		params.CertFile,
		params.KeyFile,
		params.CAFile,
	)
}
