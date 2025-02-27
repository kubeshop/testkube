package event

import "github.com/kubeshop/testkube/pkg/api/v1/testkube"

type lazyEventEmitter[T Interface] struct {
	accessor *T
}

func Lazy[T Interface](accessor *T) Interface {
	return &lazyEventEmitter[T]{accessor: accessor}
}

func (l *lazyEventEmitter[T]) Notify(event testkube.Event) {
	(*l.accessor).Notify(event)
}
