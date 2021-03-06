package scraper

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/storage/minio"
)

// Scraper is responsible for collecting and persisting the necessary artifacts
type Scraper interface {
	// Scrape gets artifacts from the directories present in the execution with executionID
	Scrape(executionID string, directories []string) error
}

// NewMinioScraper returns a Minio implementation of the Scraper
func NewMinioScraper(endpoint, accessKeyID, secretAccessKey, location, token string, ssl bool) *MinioScraper {

	return &MinioScraper{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Location:        location,
		Token:           token,
		Ssl:             ssl,
	}

}

// MinioScraper manages getting artifacts from job pods
type MinioScraper struct {
	Endpoint, AccessKeyID, SecretAccessKey, Location, Token string
	Ssl                                                     bool
}

// Scrape gets artifacts from pod based on execution ID and directories list
func (s MinioScraper) Scrape(id string, directories []string) error {
	client := minio.NewClient(s.Endpoint, s.AccessKeyID, s.SecretAccessKey, s.Location, s.Token, s.Ssl) // create storage client
	err := client.Connect()
	if err != nil {
		return fmt.Errorf("error occured creating minio client: %w", err)
	}

	return client.ScrapeArtefacts(id, directories...)
}
