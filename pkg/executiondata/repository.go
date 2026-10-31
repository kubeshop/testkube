package executiondata

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/controlplaneclient"
)

// translateUnsupported turns the bare Unimplemented a control plane returns for an
// unknown method into something that names the missing feature. Checking the
// capability up-front would not be enough on its own: a control plane that forgets
// to advertise it still has to fail comprehensibly.
func translateUnsupported(err error) error {
	if status.Code(err) != codes.Unimplemented {
		return err
	}
	return fmt.Errorf("this control plane cannot grant access to another execution's artifacts (missing the %q capability): upgrade it, or exchange the data as outputs instead: %w",
		capabilities.CapabilityArtifactRead, err)
}

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
		return nil, translateUnsupported(err)
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
