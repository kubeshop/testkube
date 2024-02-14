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
	"io"
	"io/fs"
	"os"
	"sync"

	"github.com/dustin/go-humanize"

	"github.com/kubeshop/testkube/pkg/tmp"
	"github.com/kubeshop/testkube/pkg/ui"
)

func NewTarCachedProcessor(name string, cachePath string) Processor {
	if cachePath == "" {
		cachePath = tmp.Name()
	}
	return &tarCachedProcessor{
		name:      name,
		cachePath: cachePath,
	}
}

type tarCachedProcessor struct {
	uploader  Uploader
	name      string
	cachePath string
	mu        *sync.Mutex
	errCh     chan error
	file      *os.File
	ts        *tarStream
}

func (d *tarCachedProcessor) Start() (err error) {
	d.errCh = make(chan error)
	d.mu = &sync.Mutex{}
	d.file, err = os.Create(d.cachePath)

	return err
}

func (d *tarCachedProcessor) init(uploader Uploader) {
	if d.ts != nil {
		return
	}
	d.ts = NewTarStream()
	d.uploader = uploader
	go func() {
		_, err := io.Copy(d.file, d.ts)
		d.errCh <- err
	}()
}

func (d *tarCachedProcessor) clean() {
	_ = os.Remove(d.cachePath)
}

func (d *tarCachedProcessor) upload(path string, file fs.File, stat fs.FileInfo) error {
	defer file.Close()
	return d.ts.Add(path, file, stat)
}

func (d *tarCachedProcessor) Add(uploader Uploader, path string, file fs.File, stat fs.FileInfo) error {
	d.mu.Lock()
	d.init(uploader)
	defer d.mu.Unlock()
	return d.upload(path, file, stat)
}

func (d *tarCachedProcessor) End() (err error) {
	defer d.clean()

	if d.ts != nil {
		<-d.ts.Done()
	}
	err = d.ts.Close()
	if err != nil {
		return fmt.Errorf("problem closing writer: %w", err)
	}
	err = <-d.errCh
	if err != nil {
		return fmt.Errorf("problem writing to disk cache: %w", err)
	}

	if d.uploader == nil {
		return nil
	}

	file, err := os.Open(d.cachePath)
	if err != nil {
		return fmt.Errorf("problem reading disk cache: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("problem reading disk cache: stat: %w", err)
	}

	fmt.Printf("Archived everything in %s archive (%s).\n", ui.LightCyan(d.name), ui.LightCyan(humanize.Bytes(uint64(stat.Size()))))
	return d.uploader.Add(d.name, file, stat.Size())
}
