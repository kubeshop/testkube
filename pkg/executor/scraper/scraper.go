package scraper

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// Scraper is responsible for collecting and persisting the execution artifacts
//
//go:generate mockgen -destination=./mock_scraper.go -package=scraper "github.com/kubeshop/testkube/pkg/executor/scraper" Scraper
type Scraper interface {
	// Scrape gets artifacts from the provided paths and the provided execution
	Scrape(ctx context.Context, paths []string, execution testkube.Execution) error
	Close() error
}

type ExtractLoadScraper struct {
	extractor Extractor
	loader    Uploader
}

func NewExtractLoadScraper(extractor Extractor, loader Uploader) *ExtractLoadScraper {
	return &ExtractLoadScraper{
		extractor: extractor,
		loader:    loader,
	}
}

func (s *ExtractLoadScraper) Scrape(ctx context.Context, paths []string, execution testkube.Execution) error {
	return s.
		extractor.
		Extract(ctx, paths, func(ctx context.Context, object *Object) error {
			return s.loader.Upload(ctx, object, execution)
		})
}

func (s *ExtractLoadScraper) Close() error {
	return s.loader.Close()
}
