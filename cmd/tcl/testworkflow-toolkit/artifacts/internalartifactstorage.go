// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package artifacts

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/tcl/testworkflow-toolkit/env"
)

type InternalArtifactStorage interface {
	FullPath(artifactPath string) string
	SaveStream(artifactPath string, stream io.Reader) error
	Wait() error
}

type internalArtifactStorage struct {
	prefix   string
	uploader Uploader
	startMu  sync.Mutex
	started  bool
}

func newArtifactUploader() Uploader {
	if env.CloudEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		client := env.Cloud(ctx)
		return NewCloudUploader(client, WithParallelismCloud(30), CloudDetectMimetype)
	}
	return NewDirectUploader(WithParallelism(30), DirectDetectMimetype)
}

func InternalStorage() InternalArtifactStorage {
	return &internalArtifactStorage{
		prefix:   filepath.Join(".testkube", env.Ref()),
		uploader: newArtifactUploader(),
	}
}

func (s *internalArtifactStorage) start() error {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	if s.started {
		return nil
	}
	s.started = true
	return s.uploader.Start()
}

func (s *internalArtifactStorage) FullPath(filePath string) string {
	return filepath.Join(s.prefix, filePath)
}

func (s *internalArtifactStorage) SaveStream(artifactPath string, stream io.Reader) error {
	err := s.start()
	if err != nil {
		return err
	}
	// TODO: Stream the data instead
	b, err := io.ReadAll(stream)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	buf := bytes.NewBuffer(b)
	err = s.uploader.Add(filepath.Join(s.prefix, artifactPath), buf, int64(buf.Len()))
	if err != nil {
		return err
	}
	return s.uploader.End()
}

func (s *internalArtifactStorage) Wait() error {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	if !s.started {
		return nil
	}
	return s.uploader.End()
}
