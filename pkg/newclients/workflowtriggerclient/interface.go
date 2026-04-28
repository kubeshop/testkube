package workflowtriggerclient

import (
	"context"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type ListOptions struct {
	Labels     map[string]string
	TextSearch string
	Selector   string
	Offset     uint32
	Limit      uint32
}

//go:generate go tool mockgen -destination=./mock_interface.go -package=workflowtriggerclient "github.com/kubeshop/testkube/pkg/newclients/workflowtriggerclient" WorkflowTriggerClient
type WorkflowTriggerClient interface {
	Get(ctx context.Context, environmentId string, name string, namespace string) (*testkube.WorkflowTrigger, error)
	List(ctx context.Context, environmentId string, options ListOptions, namespace string) ([]testkube.WorkflowTrigger, error)
	Update(ctx context.Context, environmentId string, trigger testkube.WorkflowTrigger) error
	Create(ctx context.Context, environmentId string, trigger testkube.WorkflowTrigger) error
	Delete(ctx context.Context, environmentId string, name string, namespace string) error
	DeleteAll(ctx context.Context, environmentId string, namespace string) (uint32, error)
	DeleteByLabels(ctx context.Context, environmentId string, selector string, namespace string) (uint32, error)
}
