package storage

import (
	"context"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"io"
)

//go:generate mockgen -destination=./artifacts_mock.go -package=storage "github.com/kubeshop/testkube/pkg/storage" ArtifactsStorage
type ArtifactsStorage interface {
	ListFiles(ctx context.Context, executionId, testName, testSuiteName string) ([]testkube.Artifact, error)
	DownloadFile(ctx context.Context, file, executionId, testName, testSuiteName string) (io.Reader, error)
}
