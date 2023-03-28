package scraper

import (
	"context"
)

// Scraper is responsible for collecting and persisting the necessary artifacts
type Scraper interface {
	// Scrape gets artifacts from the directories present in the execution with executionID
	Scrape(executionID string, directories []string) error
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

func (s *ExtractLoadScraper) Scrape(ctx context.Context, meta map[string]any) error {
	return s.
		extractor.
		Extract(ctx, func(ctx context.Context, object *Object) error {
			return s.loader.Upload(ctx, object, meta)
		})
}
