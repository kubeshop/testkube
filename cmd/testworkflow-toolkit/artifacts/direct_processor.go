package artifacts

import (
	"io/fs"
)

func NewDirectProcessor() Processor {
	return &directProcessor{}
}

type directProcessor struct {
}

func (d *directProcessor) Start() error {
	return nil
}

func (d *directProcessor) Add(uploader Uploader, path string, file fs.File, stat fs.FileInfo) error {
	return uploader.Add(path, file, stat.Size())
}

func (d *directProcessor) End() error {
	return nil
}
