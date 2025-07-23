package artifacts

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env"
	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/env/config"
	"github.com/kubeshop/testkube/pkg/bufferedstream"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

type InternalArtifactStorage interface {
	FullPath(artifactPath string) string
	SaveStream(artifactPath string, stream io.Reader) error
	SaveFile(artifactPath string, r io.Reader, info os.FileInfo) error
	Wait() error
}

type withLength interface {
	Len() int
}

type internalArtifactStorage struct {
	prefix   string
	uploader Uploader
	startMu  sync.Mutex
	started  bool
}

func newArtifactUploader() (Uploader, error) {
	cfg := config.Config()
	cloud, err := env.Cloud()
	if err != nil {
		return nil, err
	}
	return NewCloudUploader(
		cloud,
		cfg.Execution.EnvironmentId,
		cfg.Execution.Id,
		cfg.Workflow.Name,
		config.Ref(),
		WithParallelismCloud(30),
		CloudDetectMimetype,
	), nil
}

func InternalStorage() (InternalArtifactStorage, error) {
	uploader, err := newArtifactUploader()
	if err != nil {
		return nil, err
	}
	return &internalArtifactStorage{
		prefix:   filepath.Join(".testkube", config.Ref()),
		uploader: uploader,
	}, nil
}

func InternalStorageForAgent(client controlplaneclient.Client, environmentId, executionId, workflowName, ref string) InternalArtifactStorage {
	return &internalArtifactStorage{
		prefix:   filepath.Join(".testkube", ref),
		uploader: NewCloudUploader(client, environmentId, executionId, workflowName, ref, WithParallelismCloud(30), CloudDetectMimetype),
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

func (s *internalArtifactStorage) SaveFile(artifactPath string, r io.Reader, info os.FileInfo) error {
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

func (s *internalArtifactStorage) Wait() error {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	if !s.started {
		return nil
	}
	return s.uploader.End()
}
