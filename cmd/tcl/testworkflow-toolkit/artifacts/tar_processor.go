// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package artifacts

import (
	"fmt"
	"io/fs"
	"sync"

	"github.com/kubeshop/testkube/pkg/ui"
)

func NewTarProcessor(name string) Processor {
	return &tarProcessor{
		name: name,
	}
}

type tarProcessor struct {
	name  string
	mu    *sync.Mutex
	errCh chan error
	ts    *tarStream
}

func (d *tarProcessor) Start() (err error) {
	d.errCh = make(chan error)
	d.mu = &sync.Mutex{}

	return err
}

func (d *tarProcessor) init(uploader Uploader) {
	if d.ts != nil {
		return
	}
	d.ts = NewTarStream()

	// Start uploading the file
	go func() {
		err := uploader.Add(d.name, d.ts, -1)
		if err != nil {
			_ = d.ts.Close()
		}
		d.errCh <- err
	}()
}

func (d *tarProcessor) upload(path string, file fs.File, stat fs.FileInfo) error {
	defer file.Close()
	return d.ts.Add(path, file, stat)
}

func (d *tarProcessor) Add(uploader Uploader, path string, file fs.File, stat fs.FileInfo) error {
	d.mu.Lock()
	d.init(uploader)
	defer d.mu.Unlock()
	return d.upload(path, file, stat)
}

func (d *tarProcessor) End() (err error) {
	if d.ts != nil {
		<-d.ts.Done()
	}
	err = d.ts.Close()
	if err != nil {
		return fmt.Errorf("problem closing writer: %w", err)
	}

	fmt.Printf("Archived everything in %s archive.\n", ui.LightCyan(d.name))
	return <-d.errCh
}
