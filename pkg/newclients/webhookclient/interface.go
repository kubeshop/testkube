package webhookclient

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
	Resource  *testkube.Webhook
}

type Watcher channels.Watcher[Update]

//go:generate mockgen -destination=./mock_interface.go -package=webhookclient "github.com/kubeshop/testkube/pkg/newclients/webhookclient" WebhookClient
type WebhookClient interface {
	Get(ctx context.Context, environmentId string, name string) (*testkube.Webhook, error)
	GetKubernetesObjectUID(ctx context.Context, environmentId string, name string) (types.UID, error)
	List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.Webhook, error)
	ListLabels(ctx context.Context, environmentId string) (map[string][]string, error)
	Update(ctx context.Context, environmentId string, webhook testkube.Webhook) error
	UpdateStatus(ctx context.Context, environmentId string, webhook testkube.Webhook) error
	Create(ctx context.Context, environmentId string, webhook testkube.Webhook) error
	Delete(ctx context.Context, environmentId string, name string) error
	DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error)
	WatchUpdates(ctx context.Context, environmentId string, includeInitialData bool) Watcher
}
