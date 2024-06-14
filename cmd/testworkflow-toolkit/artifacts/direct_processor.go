// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

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
