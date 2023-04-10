package scraper

import (
	"context"
	"io"
)

type DataType string

const (
	// DataTypeRaw specifies that the object is a raw file
	DataTypeRaw DataType = "raw"
	// DataTypeTarball specifies that the object is a tarball (gzip compressed tar archive)
	DataTypeTarball DataType = "tarball"
)

//go:generate mockgen -destination=./mock_extractor.go -package=scraper "github.com/kubeshop/testkube/pkg/executor/scraper" Extractor
type Extractor interface {
	Extract(ctx context.Context, paths []string, process ProcessFn) error
}

type ProcessFn func(ctx context.Context, object *Object) error

type Object struct {
	Name     string
	Size     int64
	Data     io.Reader
	DataType DataType
}
