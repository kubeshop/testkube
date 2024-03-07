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
	r  io.ReadCloser
	w  io.WriteCloser
	g  io.WriteCloser
	t  *tar.Writer
	mu *sync.Mutex
	wg sync.WaitGroup
}

func NewTarStream() *tarStream {
	r, w := io.Pipe()
	g := gzip.NewWriter(w)
	t := tar.NewWriter(g)
	return &tarStream{
		r:  r,
		w:  w,
		g:  g,
		t:  t,
		mu: &sync.Mutex{},
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
	err = t.t.WriteHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(t.t, file)
	return err
}

func (t *tarStream) Read(p []byte) (n int, err error) {
	return t.r.Read(p)
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
	err = t.t.Close()
	if err != nil {
		_ = t.g.Close()
		_ = t.w.Close()
		return fmt.Errorf("closing tar: tar: %v", err)
	}
	err = t.g.Close()
	if err != nil {
		_ = t.w.Close()
		return fmt.Errorf("closing tar: gzip: %v", err)
	}
	err = t.w.Close()
	if err != nil {
		return fmt.Errorf("closing tar: pipe: %v", err)
	}
	return nil
}
