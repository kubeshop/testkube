package scraper

import (
	"context"
)

//go:generate mockgen -destination=./mock_loader.go -package=scraper "github.com/kubeshop/testkube/pkg/executor/scraper" Loader
type Loader interface {
	Load(ctx context.Context, object *Object, meta map[string]any) error
}
