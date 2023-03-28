package factory

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/agent"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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

func Scrape(ctx context.Context, dirs []string, execution testkube.Execution, params envs.Params) (err error) {
	output.PrintLog(fmt.Sprintf("%s Extracting artifacts from %s using Filesystem Extractor", ui.IconCheckMark, dirs))

	extractor := scraper.NewArchiveFilesystemExtractor(dirs, filesystem.NewOSFileSystem())

	var loader scraper.Uploader
	var meta map[string]any
	var closeF func() error
	if params.CloudMode {
		meta = cloudscraper.ExtractCloudLoaderMeta(execution)
		loader, closeF, err = getCloudLoader(ctx, params)
		if err != nil {
			return errors.Wrap(err, "error creating cloud loader")
		}
		defer closeF()
	} else {
		meta = scraper.ExtractMinIOUploaderMeta(execution)

		loader, err = getMinIOLoader(params)
		if err != nil {
			return errors.Wrap(err, "error creating minio loader")
		}
	}
	elScraper := scraper.NewExtractLoadScraper(extractor, loader)
	if err = elScraper.Scrape(ctx, meta); err != nil {
		output.PrintLog(fmt.Sprintf("%s Error encountered while scraping artifacts", ui.IconCross))
		return errors.Errorf("error scraping artifacts: %v", err)
	}

	return nil
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
	return scraper.NewMinIOLoader(
		params.Endpoint,
		params.AccessKeyID,
		params.SecretAccessKey,
		params.Location,
		params.Token,
		params.Bucket,
		params.Ssl,
	)
}
