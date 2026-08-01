package executiondata

import (
	"context"

	"github.com/kubeshop/testkube/pkg/controlplaneclient"
)

//go:generate go tool mockgen -destination=./mock_repository.go -package=executiondata "github.com/kubeshop/testkube/pkg/executiondata" ExecutionRepository

// ExecutionRepository reads data about executions this workflow did not schedule
// itself, such as its parent or an execution referenced by raw id, and the artifacts
// of any execution it may reference.
type ExecutionRepository interface {
	Get(ctx context.Context, id string) (Execution, error)
	ListArtifacts(ctx context.Context, id string, patterns []string) ([]Artifact, error)
}

// Artifact is a file an execution produced, with a temporary URL to download it from.
type Artifact struct {
	Path string
	Url  string
	Size int64
}

type executionRepository struct {
	getClient     func() controlplaneclient.Client
	environmentId string
}

func NewExecutionRepository(getClient func() controlplaneclient.Client, environmentId string) ExecutionRepository {
	return &executionRepository{getClient: getClient, environmentId: environmentId}
}

func (r *executionRepository) ListArtifacts(ctx context.Context, id string, patterns []string) ([]Artifact, error) {
	found, err := r.getClient().ListExecutionArtifactsGetPresignedURLs(ctx, r.environmentId, id, patterns)
	if err != nil {
		return nil, err
	}
	artifacts := make([]Artifact, 0, len(found))
	for _, artifact := range found {
		artifacts = append(artifacts, Artifact{Path: artifact.Path, Url: artifact.Url, Size: artifact.Size})
	}
	return artifacts, nil
}

func (r *executionRepository) Get(ctx context.Context, id string) (Execution, error) {
	execution, err := r.getClient().GetExecution(ctx, r.environmentId, id)
	if err != nil {
		return Execution{}, err
	}
	return FromExecution(execution), nil
}
