package scraper

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

//go:generate mockgen -destination=./mock_uploader.go -package=scraper "github.com/kubeshop/testkube/pkg/executor/scraper" Uploader
type Uploader interface {
	Upload(ctx context.Context, object *Object, execution testkube.Execution) error
	Close() error
}
