// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package artifacts

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"sync"
)

type tarStream struct {
	reader io.ReadCloser
	writer io.WriteCloser
	gzip   io.WriteCloser
	tar    *tar.Writer
	mu     *sync.Mutex
	wg     sync.WaitGroup
}

func NewTarStream() *tarStream {
	reader, writer := io.Pipe()
	gzip := gzip.NewWriter(writer)
	tar := tar.NewWriter(gzip)
	return &tarStream{
		reader: reader,
		writer: writer,
		gzip:   gzip,
		tar:    tar,
		mu:     &sync.Mutex{},
	}
}

func (t *tarStream) Add(path string, file fs.File, stat fs.FileInfo) error {
	t.wg.Add(1)
	t.mu.Lock()
	defer t.mu.Unlock()
	defer t.wg.Done()

	// Write file header
	name := stat.Name()
	header, err := tar.FileInfoHeader(stat, name)
	if err != nil {
		return err
	}
	header.Name = path
	err = t.tar.WriteHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(t.tar, file)
	return err
}

func (t *tarStream) Read(p []byte) (n int, err error) {
	return t.reader.Read(p)
}

func (t *tarStream) Done() chan struct{} {
	ch := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(ch)
	}()
	return ch
}

func (t *tarStream) Close() (err error) {
	err = t.tar.Close()
	if err != nil {
		_ = t.gzip.Close()
		_ = t.writer.Close()
		return fmt.Errorf("closing tar: tar: %v", err)
	}
	err = t.gzip.Close()
	if err != nil {
		_ = t.writer.Close()
		return fmt.Errorf("closing tar: gzip: %v", err)
	}
	err = t.writer.Close()
	if err != nil {
		return fmt.Errorf("closing tar: pipe: %v", err)
	}
	return nil
}
