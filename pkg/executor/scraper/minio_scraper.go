package scraper

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/ui"
)

// NewMinioScraper returns a Minio implementation of the Scraper
func NewMinioScraper(endpoint, accessKeyID, secretAccessKey, region, token, bucket string, ssl bool) *MinioScraper {

	return &MinioScraper{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Region:          region,
		Token:           token,
		Bucket:          bucket,
		Ssl:             ssl,
	}

}

// MinioScraper manages getting artifacts from job pods
type MinioScraper struct {
	Endpoint, AccessKeyID, SecretAccessKey, Region, Token, Bucket string
	Ssl                                                           bool
}

// Scrape gets artifacts from pod based on execution ID and directories list
func (s MinioScraper) Scrape(ctx context.Context, directories []string, execution testkube.Execution) error {
	output.PrintLog(fmt.Sprintf("%s Scraping artifacts %s", ui.IconCabinet, directories))
	client := minio.NewClient(s.Endpoint, s.AccessKeyID, s.SecretAccessKey, s.Region, s.Token, s.Bucket, s.Ssl) // create storage client
	err := client.Connect()
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to scrape artifacts: %s", ui.IconCross, err.Error()))
		return fmt.Errorf("error occured creating minio client: %w", err)
	}

	err = client.ScrapeArtefacts(ctx, execution.Id, directories...)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to scrape artifacts: %s", ui.IconCross, err.Error()))
		return err
	}

	output.PrintLog(fmt.Sprintf("%s Successfully scraped artifacts", ui.IconCheckMark))
	return nil
}

var _ Scraper = (*MinioScraper)(nil)
