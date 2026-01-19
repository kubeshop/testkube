package artifacts

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/pkg/bufferedstream"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

// StorageProvider allows injection of different storage implementations
type StorageProvider interface {
	GetUploader(environmentId, executionId, workflowName, ref string) (Uploader, error)
}

// CloudStorageProvider uses the real Cloud connection
type CloudStorageProvider struct{}

func (c *CloudStorageProvider) GetUploader(environmentId, executionId, workflowName, ref string) (Uploader, error) {
	cloud, err := env.Cloud()
	if err != nil {
		return nil, err
	}
	return NewCloudUploader(
		cloud,
		environmentId,
		executionId,
		workflowName,
		ref,
		WithParallelismCloud(30),
		CloudDetectMimetype,
	), nil
}

// NoOpStorageProvider discards all artifacts (for testing zero-storage scenarios)
type NoOpStorageProvider struct{}

func (n *NoOpStorageProvider) GetUploader(environmentId, executionId, workflowName, ref string) (Uploader, error) {
	return &noOpUploader{}, nil
}

type noOpUploader struct{}

func (n *noOpUploader) Start() error { return nil }
func (n *noOpUploader) Add(path string, reader io.Reader, size int64) error {
	// Consume the reader to avoid blocking
	io.Copy(io.Discard, reader)
	return nil
}
func (n *noOpUploader) End() error { return nil }

// internalArtifactStorageV2 implements the new flexible storage
type internalArtifactStorageV2 struct {
	prefix   string
	uploader Uploader
	startMu  sync.Mutex
	started  bool
}

func (s *internalArtifactStorageV2) start() error {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	if s.started {
		return nil
	}
	s.started = true
	return s.uploader.Start()
}

func (s *internalArtifactStorageV2) FullPath(filePath string) string {
	return filepath.Join(s.prefix, filePath)
}

func (s *internalArtifactStorageV2) SaveStream(artifactPath string, stream io.Reader) error {
	err := s.start()
	if err != nil {
		return err
	}

	var size int
	if streamL, ok := stream.(withLength); ok {
		size = streamL.Len()
	} else {
		stream, err = bufferedstream.NewBufferedStream(constants.DefaultTmpDirPath, "log", stream)
		if err != nil {
			return err
		}
		defer stream.(bufferedstream.BufferedStream).Cleanup()
		size = stream.(bufferedstream.BufferedStream).Len()
	}
	err = s.uploader.Add(filepath.Join(s.prefix, artifactPath), stream, int64(size))
	if err != nil {
		return err
	}
	return s.uploader.End()
}

func (s *internalArtifactStorageV2) SaveFile(artifactPath string, r io.Reader, info os.FileInfo) error {
	if info.IsDir() {
		return errors.Errorf("error saving file: %q is a directory", info.Name())
	}
	err := s.start()
	if err != nil {
		return err
	}

	if err = s.uploader.Add(filepath.Join(s.prefix, artifactPath), r, info.Size()); err != nil {
		return err
	}
	return s.uploader.End()
}

func (s *internalArtifactStorageV2) Wait() error {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	if !s.started {
		return nil
	}
	return s.uploader.End()
}
