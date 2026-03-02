package testtriggerclient

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

//go:generate go tool mockgen -destination=./mock_interface.go -package=testtriggerclient "github.com/kubeshop/testkube/pkg/newclients/testtriggerclient" TestTriggerClient
type TestTriggerClient interface {
	Get(ctx context.Context, environmentId string, name string, namespace string) (*testkube.TestTrigger, error)
	List(ctx context.Context, environmentId string, options ListOptions, namespace string) ([]testkube.TestTrigger, error)
	Update(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error
	Create(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error
	Delete(ctx context.Context, environmentId string, name string, namespace string) error
	DeleteAll(ctx context.Context, environmentId string, namespace string) (uint32, error)
	DeleteByLabels(ctx context.Context, environmentId string, selector string, namespace string) (uint32, error)
}
