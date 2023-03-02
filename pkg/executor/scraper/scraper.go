package scraper

import (
	"context"
)

// Scraper is responsible for collecting and persisting the necessary artifacts
type Scraper interface {
	// Scrape gets artifacts from the directories present in the execution with executionID
	Scrape(executionID string, directories []string) error
}

type ScraperV2 struct {
	extractor Extractor
	loader    Uploader
}

func NewScraperV2(extractor Extractor, loader Uploader) *ScraperV2 {
	return &ScraperV2{
		extractor: extractor,
		loader:    loader,
	}
}

func (s *ScraperV2) Scrape(ctx context.Context, meta map[string]any) error {
	return s.
		extractor.
		Extract(ctx, func(ctx context.Context, object *Object) error {
			return s.loader.Upload(ctx, object, meta)
		})
}
