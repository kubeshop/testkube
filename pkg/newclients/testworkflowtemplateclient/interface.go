package testworkflowtemplateclient

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

//go:generate mockgen -destination=./mock_interface.go -package=testworkflowtemplateclient "github.com/kubeshop/testkube/pkg/newclients/testworkflowtemplateclient" TestWorkflowTemplateClient
type TestWorkflowTemplateClient interface {
	Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflowTemplate, error)
	List(ctx context.Context, environmentId string, labels map[string]string) ([]testkube.TestWorkflowTemplate, error)
	ListLabels(ctx context.Context, environmentId string) (map[string][]string, error)
	Update(ctx context.Context, environmentId string, template testkube.TestWorkflowTemplate) error
	Create(ctx context.Context, environmentId string, template testkube.TestWorkflowTemplate) error
	Delete(ctx context.Context, environmentId string, name string) error
	DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) error
}
