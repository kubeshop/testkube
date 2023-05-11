package scraper

import (
	"bytes"
	"context"
	"fmt"
	"io"

	cdevents "github.com/cdevents/sdk-go/pkg/api"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/gabriel-vasile/mimetype"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	cde "github.com/kubeshop/testkube/pkg/mapper/cdevents"
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
	extractor      Extractor
	loader         Uploader
	cdeventsClient cloudevents.Client
	clusterID      string
}

func NewExtractLoadScraper(extractor Extractor, loader Uploader, cdeventsClient cloudevents.Client, clusterID string) *ExtractLoadScraper {
	return &ExtractLoadScraper{
		extractor:      extractor,
		loader:         loader,
		cdeventsClient: cdeventsClient,
		clusterID:      clusterID,
	}
}

func (s *ExtractLoadScraper) Scrape(ctx context.Context, paths []string, execution testkube.Execution) error {
	return s.
		extractor.
		Extract(ctx, paths, func(ctx context.Context, object *Object) error {
			if s.cdeventsClient != nil {
				if err := s.sendCDEvent(execution, object); err != nil {
					log.DefaultLogger.Warnf("failing to send cd event %w", err)
				}
			}

			return s.loader.Upload(ctx, object, execution)
		})
}

func (s *ExtractLoadScraper) Close() error {
	return s.loader.Close()
}

func (s *ExtractLoadScraper) sendCDEvent(execution testkube.Execution, object *Object) error {
	header := bytes.NewBuffer(nil)
	mtype, err := mimetype.DetectReader(io.TeeReader(object.Data, header))
	if err != nil {
		log.DefaultLogger.Warnf("failing to detect mime type %w", err)
	}

	object.Data = io.MultiReader(header, object.Data)

	ev, err := cde.MapTestkubeArtifactToCDEvent(&execution, s.clusterID, "report", mtype.String())
	if err != nil {
		return err
	}

	ce, err := cdevents.AsCloudEvent(ev)
	if err != nil {
		return err
	}

	if result := s.cdeventsClient.Send(context.Background(), *ce); cloudevents.IsUndelivered(result) {
		return fmt.Errorf("failed to send, %v", result)
	}

	return nil
}
