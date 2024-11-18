package executionworkertypes

import (
	"sync/atomic"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type channelWatcher[T any] struct {
	ch       chan T
	finished atomic.Bool
	err      atomic.Value
}

func (n *channelWatcher[T]) Send(notification T) {
	n.ch <- notification
}

func (n *channelWatcher[T]) Close(err error) {
	if n.finished.CompareAndSwap(false, true) {
		if err != nil {
			n.err.Store(err)
		}
		close(n.ch)
	}
}

func (n *channelWatcher[T]) Channel() <-chan T {
	return n.ch
}

func (n *channelWatcher[T]) Err() error {
	err := n.err.Load()
	if err == nil {
		return nil
	}
	return err.(error)
}

func newChannelWatcher[T any]() *channelWatcher[T] {
	return &channelWatcher[T]{ch: make(chan T)}
}

func NewNotificationsWatcher() WritableNotificationsWatcher {
	return newChannelWatcher[testkube.TestWorkflowExecutionNotification]()
}

type NotificationsWatcher interface {
	Channel() <-chan testkube.TestWorkflowExecutionNotification
	Err() error
}

type WritableNotificationsWatcher interface {
	NotificationsWatcher
	Send(notification testkube.TestWorkflowExecutionNotification)
	Close(err error)
}

func NewStatusNotificationsWatcher() WritableStatusNotificationsWatcher {
	return newChannelWatcher[StatusNotification]()
}

type StatusNotificationsWatcher interface {
	Channel() <-chan StatusNotification
	Err() error
}

type WritableStatusNotificationsWatcher interface {
	StatusNotificationsWatcher
	Send(notification StatusNotification)
	Close(err error)
}
