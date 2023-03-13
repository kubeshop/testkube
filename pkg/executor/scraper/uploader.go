package scraper

import (
	"context"
)

//go:generate mockgen -destination=./mock_uploader.go -package=scraper "github.com/kubeshop/testkube/pkg/executor/scraper" Uploader
type Uploader interface {
	Upload(ctx context.Context, object *Object, meta map[string]any) error
}
