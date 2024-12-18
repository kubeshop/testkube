package testworkflowclient

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type ListOptions struct {
	Labels     map[string]string
	TextSearch string
	Offset     uint32
	Limit      uint32
}

//go:generate mockgen -destination=./mock_interface.go -package=testworkflowclient "github.com/kubeshop/testkube/pkg/newclients/testworkflowclient" TestWorkflowClient
type TestWorkflowClient interface {
	Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflow, error)
	List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.TestWorkflow, error)
	ListLabels(ctx context.Context, environmentId string) (map[string][]string, error)
	Update(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error
	Create(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error
	Delete(ctx context.Context, environmentId string, name string) error
	DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error)
}
