package testtriggerclient

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/repository/channels"
)

type ListOptions struct {
	Labels     map[string]string
	TextSearch string
	Selector   string
	Offset     uint32
	Limit      uint32
}

type EventType string

const (
	EventTypeCreate EventType = "create"
	EventTypeUpdate EventType = "update"
	EventTypeDelete EventType = "delete"
)

type Update struct {
	Type      EventType
	Timestamp time.Time
	Resource  *testkube.TestTrigger
}

type Watcher channels.Watcher[Update]

//go:generate mockgen -destination=./mock_interface.go -package=testtriggerclient "github.com/kubeshop/testkube/pkg/newclients/testtriggerclient" TestTriggerClient
type TestTriggerClient interface {
	Get(ctx context.Context, environmentId string, name string, namespace string) (*testkube.TestTrigger, error)
	GetKubernetesObjectUID(ctx context.Context, environmentId string, name string, namespace string) (types.UID, error)
	List(ctx context.Context, environmentId string, options ListOptions, namespace string) ([]testkube.TestTrigger, error)
	ListLabels(ctx context.Context, environmentId string, namespace string) (map[string][]string, error)
	Update(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error
	Create(ctx context.Context, environmentId string, trigger testkube.TestTrigger) error
	Delete(ctx context.Context, environmentId string, name string, namespace string) error
	DeleteAll(ctx context.Context, environmentId string, namespace string) (uint32, error)
	DeleteByLabels(ctx context.Context, environmentId string, selector string, namespace string) (uint32, error)
	WatchUpdates(ctx context.Context, environmentId string, namespace string, includeInitialData bool) Watcher
}
