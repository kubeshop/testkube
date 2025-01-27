package testworkflowclient

import (
	"context"
	"time"

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
	Resource  *testkube.TestWorkflow
}

type Watcher channels.Watcher[Update]

//go:generate mockgen -destination=./mock_interface.go -package=testworkflowclient "github.com/kubeshop/testkube/pkg/newclients/testworkflowclient" TestWorkflowClient
type TestWorkflowClient interface {
	Get(ctx context.Context, environmentId string, name string) (*testkube.TestWorkflow, error)
	List(ctx context.Context, environmentId string, options ListOptions) ([]testkube.TestWorkflow, error)
	ListLabels(ctx context.Context, environmentId string) (map[string][]string, error)
	Update(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error
	Create(ctx context.Context, environmentId string, workflow testkube.TestWorkflow) error
	Delete(ctx context.Context, environmentId string, name string) error
	DeleteByLabels(ctx context.Context, environmentId string, labels map[string]string) (uint32, error)
	WatchUpdates(ctx context.Context, environmentId string, includeInitialData bool) Watcher
}
