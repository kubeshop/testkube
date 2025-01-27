package crdstorage

import (
	"context"
	"time"

	"github.com/kubeshop/testkube/pkg/repository/channels"
	"github.com/kubeshop/testkube/pkg/resourcepattern"
)

type EventType string

const (
	EventTypeCreate EventType = "create"
	EventTypeUpdate EventType = "update"
	EventTypeDelete EventType = "delete"
)

type Resource[T any] struct {
	Resource T
	Metadata resourcepattern.Metadata
}

type Event[T any] struct {
	Type      EventType
	Timestamp time.Time
	Resource  T
	Metadata  resourcepattern.Metadata
}

type WritableStorage[T any] interface {
	Process(ctx context.Context, event Event[T]) error
}

type ReadableStorage[T any] interface {
	List(ctx context.Context) channels.Watcher[Resource[T]]
	Watch(ctx context.Context) channels.Watcher[Event[T]]
}

type Storage[T any] interface {
	WritableStorage[T]
	ReadableStorage[T]
}
