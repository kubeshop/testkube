// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package artifacts

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	minio2 "github.com/minio/minio-go/v7"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/pkg/storage/minio"
	"github.com/kubeshop/testkube/pkg/ui"
)

type PutObjectOptionsEnhancer = func(options *minio2.PutObjectOptions, path string, size int64)

func NewDirectUploader(opts ...DirectUploaderOpt) Uploader {
	uploader := &directUploader{
		parallelism: 1,
		options:     make([]PutObjectOptionsEnhancer, 0),
	}
	for _, opt := range opts {
		opt(uploader)
	}
	return uploader
}

type directUploader struct {
	client      *minio.Client
	wg          sync.WaitGroup
	sema        chan struct{}
	parallelism int
	error       atomic.Bool
	options     []PutObjectOptionsEnhancer
}

func (d *directUploader) Start() (err error) {
	d.client, err = env.ObjectStorageClient()
	d.sema = make(chan struct{}, d.parallelism)
	return err
}

func (d *directUploader) buildOptions(path string, size int64) (options minio2.PutObjectOptions) {
	for _, enhance := range d.options {
		enhance(&options, path, size)
	}
	if options.ContentType == "" {
		options.ContentType = "application/octet-stream"
	}
	return options
}

func (d *directUploader) upload(path string, file io.ReadCloser, size int64) {
	ns := env.ExecutionId()
	opts := d.buildOptions(path, size)
	err := d.client.SaveFileDirect(context.Background(), ns, path, file, size, opts)

	if err != nil {
		d.error.Store(true)
		ui.Errf("%s: failed: %s", path, err.Error())
		return
	}
}

func (d *directUploader) Add(path string, file io.ReadCloser, size int64) error {
	d.wg.Add(1)
	d.sema <- struct{}{}
	go func() {
		d.upload(path, file, size)
		_ = file.Close()
		d.wg.Done()
		<-d.sema
	}()
	return nil
}

func (d *directUploader) End() error {
	d.wg.Wait()
	if d.error.Load() {
		return fmt.Errorf("upload failed")
	}
	return nil
}

type DirectUploaderOpt = func(uploader *directUploader)

func WithParallelism(parallelism int) DirectUploaderOpt {
	return func(uploader *directUploader) {
		if parallelism < 1 {
			parallelism = 1
		}
		uploader.parallelism = parallelism
	}
}

func WithMinioOptionsEnhancer(fn PutObjectOptionsEnhancer) DirectUploaderOpt {
	return func(uploader *directUploader) {
		uploader.options = append(uploader.options, fn)
	}
}
