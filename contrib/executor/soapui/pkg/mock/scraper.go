package mock

import "log"

// Scraper implements a mock for the Scraper from "github.com/kubeshop/testkube/pkg/executor/scraper"
type Scraper struct {
	ScrapeFn func(id string, directories []string) error
}

func (s Scraper) Scrape(id string, directories []string) error {
	if s.ScrapeFn == nil {
		log.Fatal("not implemented")
	}
	return s.ScrapeFn(id, directories)
}
