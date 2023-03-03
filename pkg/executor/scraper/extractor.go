package scraper

import (
	"context"
	"io"
)

//go:generate mockgen -destination=./mock_extractor.go -package=scraper "github.com/kubeshop/testkube/pkg/executor/scraper" Extractor
type Extractor interface {
	Extract(ctx context.Context, process ProcessFn) error
}

type ProcessFn func(ctx context.Context, object *Object) error

type Object struct {
	Name string
	Size int64
	Data io.Reader
}
