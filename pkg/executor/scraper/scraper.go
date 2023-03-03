package scraper

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/ui"
)

// Scraper is responsible for collecting and persisting the necessary artifacts
type Scraper interface {
	// Scrape gets artifacts from the directories present in the execution with executionID
	Scrape(executionID string, directories []string) error
}

// NewMinioScraper returns a Minio implementation of the Scraper
func NewMinioScraper(endpoint, accessKeyID, secretAccessKey, location, token, bucket string, ssl bool) *MinioScraper {

	return &MinioScraper{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Location:        location,
		Token:           token,
		Bucket:          bucket,
		Ssl:             ssl,
	}

}

// MinioScraper manages getting artifacts from job pods
type MinioScraper struct {
	Endpoint, AccessKeyID, SecretAccessKey, Location, Token, Bucket string
	Ssl                                                             bool
}

// Scrape gets artifacts from pod based on execution ID and directories list
func (s MinioScraper) Scrape(id string, directories []string) error {
	output.PrintLog(fmt.Sprintf("%s Scraping artifacts %s", ui.IconCabinet, directories))
	client := minio.NewClient(s.Endpoint, s.AccessKeyID, s.SecretAccessKey, s.Location, s.Token, s.Bucket, s.Ssl) // create storage client
	err := client.Connect()
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to scrape artifacts: %s", ui.IconCross, err.Error()))
		return fmt.Errorf("error occured creating minio client: %w", err)
	}

	err = client.ScrapeArtefacts(id, directories...)
	if err != nil {
		output.PrintLog(fmt.Sprintf("%s Failed to scrape artifacts: %s", ui.IconCross, err.Error()))
		return err
	}

	output.PrintLog(fmt.Sprintf("%s Successfully scraped artifacts", ui.IconCheckMark))
	return nil
}
