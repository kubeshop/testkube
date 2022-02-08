package scrapper

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/storage/minio"
)

func NewScrapper(endpoint, accessKeyID, secretAccessKey, location, token string, ssl bool) *Scrapper {

	return &Scrapper{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Location:        location,
		Token:           token,
		Ssl:             ssl,
	}

}

type Scrapper struct {
	Endpoint, AccessKeyID, SecretAccessKey, Location, Token string
	Ssl                                                     bool
}

func (s Scrapper) Scrape(id string, directories []string) error {
	client := minio.NewClient(s.Endpoint, s.AccessKeyID, s.SecretAccessKey, s.Location, s.Token, s.Ssl) // create storage client
	err := client.Connect()
	if err != nil {
		return fmt.Errorf("error occured creating minio client: %w", err)
	}

	return client.ScrapeArtefacts(id, directories...)
}
