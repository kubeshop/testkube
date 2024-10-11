package executionworker

import (
	"sync/atomic"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type channelWatcher[T any] struct {
	ch       chan T
	finished atomic.Bool
	err      atomic.Value
}

func (n *channelWatcher[T]) send(notification T) {
	n.ch <- notification
}

func (n *channelWatcher[T]) close(err error) {
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

func newNotificationsWatcher() *channelWatcher[testkube.TestWorkflowExecutionNotification] {
	return newChannelWatcher[testkube.TestWorkflowExecutionNotification]()
}

type NotificationsWatcher interface {
	Channel() <-chan testkube.TestWorkflowExecutionNotification
	Err() error
}

func newStatusNotificationsWatcher() *channelWatcher[StatusNotification] {
	return newChannelWatcher[StatusNotification]()
}

type StatusNotificationsWatcher interface {
	Channel() <-chan StatusNotification
	Err() error
}
