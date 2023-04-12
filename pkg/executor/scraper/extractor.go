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

type FilesMeta struct {
	// DataType is the type of data that is stored
	DataType DataType `json:"dataType"`
	// Files is a list of files that are stored and their original sizes
	Files []*FileStat `json:"files"`
	// Archive is the name of the archive file that contains all the files
	Archive string `json:"archive,omitempty"`
}
type FileStat struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}
