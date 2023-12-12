package scraper

import (
	"context"
	"fmt"

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
	Scrape(ctx context.Context, paths, masks []string, execution testkube.Execution) error
	Close() error
}

type ExtractLoadScraper struct {
	extractor      Extractor
	loader         Uploader
	cdeventsClient cloudevents.Client
	clusterID      string
	dashboardURI   string
}

func NewExtractLoadScraper(extractor Extractor, loader Uploader, cdeventsClient cloudevents.Client,
	clusterID, dashboardURI string) *ExtractLoadScraper {
	return &ExtractLoadScraper{
		extractor:      extractor,
		loader:         loader,
		cdeventsClient: cdeventsClient,
		clusterID:      clusterID,
		dashboardURI:   dashboardURI,
	}
}

func (s *ExtractLoadScraper) Scrape(ctx context.Context, paths, masks []string, execution testkube.Execution) error {
	return s.
		extractor.
		Extract(ctx, paths, masks,
			func(ctx context.Context, object *Object) error {
				return s.loader.Upload(ctx, object, execution)
			},
			func(ctx context.Context, path string) error {
				if s.cdeventsClient != nil {
					if err := s.sendCDEvent(execution, path); err != nil {
						return err
					}
				}

				return nil
			})
}

func (s *ExtractLoadScraper) Close() error {
	return s.loader.Close()
}

func (s *ExtractLoadScraper) sendCDEvent(execution testkube.Execution, path string) error {
	mtype, err := mimetype.DetectFile(path)
	if err != nil {
		log.DefaultLogger.Warnf("failed to detect mime type %w", err)
	}

	ev, err := cde.MapTestkubeArtifactToCDEvent(&execution, s.clusterID, path, mtype.String(), s.dashboardURI)
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
